package main

import (
	"bytes"
	"container/ring"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/golang/glog"
	"github.com/DataDog/datadog-go/statsd"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"io/ioutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"
)

var (
	duration = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name: "request_duration_seconds",
			Help: "Summary of request durations.",
		},
		[]string{"label"},
	)
	counter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "request_count",
			Help: "Count of requests",
		},
		[]string{"label"},
	)
)

type ResourceWork interface {
	Run() error
}

type CPURequest struct {
	Cycles int  `form:"cycles" json:"cycles"`
	Noise  *int `form:"noise" json:"noise,omitempty"` // in percentage
	Rounds *int `form:"rounds" json:"rounds,omitempty"`
}

type MemRequest struct {
	Size   int  `form:"size" json:"size"` // in MB
	Rounds *int `form:"rounds" json:"rounds,omitempty"`
}

type NetworkRequest struct {
	Bandwidth int  `form:"bandwidth" json:"bandwidth"` // in mbps
	Rounds    *int `form:"rounds" json:"rounds,omitempty"`
}

type BlkIoRequest struct {
	ReadSize  int  `form:"readSize" json:"readSize,omitempty"`   // in bytes
	WriteSize int  `form:"writeSize" json:"writeSize,omitempty"` // in bytes
	Rounds    *int `form:"rounds" json:"rounds,omitempty"`
}

type ResourceRequest struct {
	CPURequest     *CPURequest     `form:"cpu" json:"cpu,omitempty"`
	MemRequest     *MemRequest     `form:"mem" json:"mem,omitempty"`
	NetworkRequest *NetworkRequest `form:"network" json:"network,omitempty"`
	BlkIoRequest   *BlkIoRequest   `from:"blkio" json:"blkio,omitempty"`
}

type ResourceRequestHandler struct {
	KubeClient   *kubernetes.Clientset
	NetworkPeers *ring.Ring
}

func (handler *ResourceRequestHandler) GetKubeClient() error {
	config, err := rest.InClusterConfig()
	if err != nil {
		return errors.New("Failed to load in-cluster kubeconfig: " + err.Error())
	}

	handler.KubeClient, err = kubernetes.NewForConfig(config)
	if err != nil {
		return errors.New("Failed to initialize Kubernetes client: " + err.Error())
	}

	return nil
}

func (handler *ResourceRequestHandler) GetNetworkPeers() error {
	currentPod := os.Getenv("HOSTNAME")

	for {
		statefulSet, _ := handler.KubeClient.AppsV1beta1().StatefulSets("default").Get(
			"resource-worker",
			metav1.GetOptions{},
		)
		if statefulSet.Status.ReadyReplicas == *statefulSet.Spec.Replicas {
			break
		} else {
			time.Sleep(3)
		}
	}

	pods, err := handler.KubeClient.CoreV1().Pods("default").List(
		metav1.ListOptions{LabelSelector: "app=resource-worker"},
	)

	if err != nil {
		return errors.New("Failed to list available pods: " + err.Error())
	}

	handler.NetworkPeers = ring.New(len(pods.Items) - 1)

	for _, pod := range pods.Items {
		podName := pod.GetName()
		if podName != currentPod {
			handler.NetworkPeers.Value = fmt.Sprintf(
				"%s.%s.%s.svc.cluster.local",
				podName,
				pod.Spec.Subdomain,
				pod.Namespace,
			)
			handler.NetworkPeers = handler.NetworkPeers.Next()
		}
	}

	return nil
}

func (handler *ResourceRequestHandler) RunCPURequest(request *CPURequest) error {
	if request.Rounds == nil {
		defaultRounds := 1
		request.Rounds = &defaultRounds
	}
	if request.Noise == nil {
		defaultNoise := 0
		request.Noise = &defaultNoise
	}

	ratio := 1.0 + (rand.Float32()-0.5)*2.0*float32(*request.Noise)/100.0
	for r := 0; r < *request.Rounds; r++ {
		for i := 0; i < int(float32(request.Cycles)*ratio); i++ {
		}
	}

	return nil
}

func (handler *ResourceRequestHandler) RunMemRequest(request *MemRequest) error {
	if request.Rounds == nil {
		defaultRounds := 1
		request.Rounds = &defaultRounds
	}

	arraySize := request.Size * 1024 * 1024 / 8
	dataArray := make([]int64, uint64(arraySize))
	if len(dataArray) != arraySize {
		return errors.New("Unable to allocate an array of " + strconv.Itoa(arraySize*8) + " Bytes")
	}
	fmt.Println("[Info] Successfully allocated an array of " + strconv.Itoa(arraySize*8) + " Bytes")

	for r := 0; r < *request.Rounds; r++ {
		for i := 0; i < len(dataArray); i++ {
			dataArray[i] += 1.0 / int64(request.Size)
		}
	}

	return nil
}

func (handler *ResourceRequestHandler) RunNetworkRequest(request *NetworkRequest) error {
	if request.Rounds == nil {
		defaultRounds := 1
		request.Rounds = &defaultRounds
	}

	url := fmt.Sprintf("http://%s:7998/network-endpoint", handler.NetworkPeers.Value)
	handler.NetworkPeers = handler.NetworkPeers.Next()

	content := make([]byte, request.Bandwidth)
	for r := 0; r < *request.Rounds; r++ {
		resp, err := http.Post(url, "text/plain", bytes.NewReader(content))

		if err != nil {
			return errors.New("Failed to make POST request: " + err.Error())
		} else if resp.StatusCode != http.StatusOK {
			return errors.New(fmt.Sprintf("Return code %d", resp.StatusCode))
		}
	}
	return nil
}

func (handler *ResourceRequestHandler) RunBlkIoRequest(request *BlkIoRequest) error {
	if request.Rounds == nil {
		defaultRounds := 1
		request.Rounds = &defaultRounds
	}

	if request.ReadSize != 0 {
		readerArray := make([]byte, request.ReadSize)
		file, err := os.Open("testfile")
		if err != nil {
			return errors.New(fmt.Sprintf("Failed to open file: %s", err.Error()))
		}
		fileStat, err := file.Stat()
		if err != nil {
			return errors.New(fmt.Sprintf("Failed to get file stat: %s", err.Error()))
		}

		for i := 0; i < *request.Rounds; i ++ {
			readAt := rand.Int63n(fileStat.Size() - int64(request.ReadSize))
			if _, err := file.ReadAt(readerArray, readAt); err != nil {
				return errors.New(fmt.Sprintf("Failed to read file: %s", err.Error()))
			}
		}

		if err := file.Close(); err != nil {
			return errors.New(fmt.Sprintf("Failed to close file: %s", err.Error()))
		}
	}

	if request.WriteSize != 0 {
		for i := 0; i < *request.Rounds; i++ {
			tmpfile, err := ioutil.TempFile("", "rws")
			if err != nil {
				return errors.New(fmt.Sprintf("Failed to create temp file: %s", err.Error()))
			}
			defer os.Remove(tmpfile.Name())

			content := make([]byte, request.WriteSize)
			if _, err := tmpfile.Write(content); err != nil {
				return errors.New(fmt.Sprintf("Failed to write into file: %s", err.Error()))
			}

			if err := tmpfile.Close(); err != nil {
				return errors.New(fmt.Sprintf("Failed to close file: %s", err.Error()))
			}
		}
	}

	return nil
}

func (handler *ResourceRequestHandler) Run(request *ResourceRequest) error {
	numRuns := 0
	if request.CPURequest != nil {
		if err := handler.RunCPURequest(request.CPURequest); err != nil {
			return errors.New(fmt.Sprintf("CPU request failure: %s", err.Error()))
		}
		numRuns++
	}

	if request.MemRequest != nil {
		if err := handler.RunMemRequest(request.MemRequest); err != nil {
			return errors.New(fmt.Sprintf("Memory request failure: %s", err.Error()))
		}
		numRuns++
	}

	if request.NetworkRequest != nil {
		if err := handler.RunNetworkRequest(request.NetworkRequest); err != nil {
			return errors.New(fmt.Sprintf("Network request failure: %s", err.Error()))
		}
		numRuns++
	}

	if request.BlkIoRequest != nil {
		if err := handler.RunBlkIoRequest(request.BlkIoRequest); err != nil {
			return errors.New(fmt.Sprintf("BlkIO request failure: %s", err.Error()))
		}
		numRuns++
	}

	if numRuns == 0 {
		return errors.New("Requested resource not recognized or implemented")
	}

	return nil
}

type Work struct {
	Requests []ResourceRequest `form:"requests" json:"requests"`
	Label    string            `form:"label" json:"label,omitempty"`
}


type StatsdPublisher struct {
	Client *statsd.Client
}

func NewStatsdPublisher() (*StatsdPublisher, error) {
	publisher := &StatsdPublisher{}

	client, err := statsd.New(fmt.Sprintf("%s:8125", os.Getenv("NODE_NAME")))

	if err != nil {
		fmt.Errorf(err.Error())
		return nil, errors.New(fmt.Sprintf("Failed to create dogstatsd client: %s", err.Error()))
	}

	publisher.Client = client
	publisher.Client.Namespace = "hyperpilot.resource-worker-service."

	return publisher, nil
}

func (publisher *StatsdPublisher) Incr(metric string, tags []string) error {
	return publisher.Client.Incr(metric, tags, 1)
}

func (publisher *StatsdPublisher) Timing(metric string, duration time.Duration, tags []string) error {
	return publisher.Client.Timing(metric, duration, tags, 1)
}

func main() {

	prometheus.MustRegister(duration)
	prometheus.MustRegister(counter)

	statsdPub, err := NewStatsdPublisher()
	if err != nil {
		panic(err)
	}

	handler := &ResourceRequestHandler{}
	if err := handler.GetKubeClient(); err != nil {
		fmt.Println("[Warning] Cannot get Kube Client; no support for network request!")
	} else if err := handler.GetNetworkPeers(); err != nil {
		fmt.Println("[Warning] Cannot get network peers; no support for network request!")
	}

	r := gin.Default()

	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	r.POST("/network-endpoint", func(c *gin.Context) {
		body, err := ioutil.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   err.Error(),
			})
		}
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"length":  len(body),
		})
	})

	r.POST("/run", func(c *gin.Context) {

		var work Work
		if err := c.BindJSON(&work); err == nil {

			startTime := time.Now()
			defer func(begun time.Time) {
				promLabels := prometheus.Labels{"label": work.Label}
				timePassed := time.Since(begun)

				duration.With(promLabels).Observe(timePassed.Seconds())
				counter.With(promLabels).Inc()

				statsdLabels := []string{fmt.Sprintf("label:%s", work.Label)}
				statsdPub.Timing("request_duration", timePassed, statsdLabels)
				statsdPub.Incr("request_count", statsdLabels)
			}(startTime)

			// Run the worker
			for i, request := range work.Requests {
				if err := handler.Run(&request); err != nil {
					message := fmt.Sprintf("Request no. %d failed: %s", i, err)
					glog.Errorf("%s", message)
					c.JSON(http.StatusInternalServerError, gin.H{
						"success": false,
						"reason":  message,
					})
					return
				}
			}

			c.JSON(http.StatusOK, gin.H{
				"success":  true,
				"duration": time.Since(startTime).Seconds(),
				// Put the results / summaries of the work in response
			})
		} else {
			// Handle error
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"reason":  "Invalid input data",
			})
		}
	})

	r.Run(":7998")
}
