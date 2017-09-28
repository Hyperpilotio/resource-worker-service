package main

import (
	"os"
	"fmt"
	"time"
	"bytes"
	"errors"
	"net/http"
	"io/ioutil"
	"container/ring"
	"github.com/gin-gonic/gin"
	// "github.com/golang/glog"
	"k8s.io/client-go/kubernetes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	duration = prometheus.NewSummary(prometheus.SummaryOpts{
		Name: "request_duration_seconds",
		Help: "Summary of request durations.",
	})
	counter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "request_count",
			Help: "Count of requests",
		},
		[]string{"resource"},
	)
)

type ResourceWork interface {
	Run() error
}

type CPURequest struct {
	Cycles int `form:"cycles" json:"cycles"`
}

type NetworkRequest struct {
	Bandwidth int `form:"bandwidth" json:"bandwidth"`
}

type BlkIoRequest struct {
}

type ResourceRequest struct {
	CPURequest     *CPURequest     `form:"cpu" json:"cpu,omitempty"`
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
		return err
	}

	handler.KubeClient, err = kubernetes.NewForConfig(config)
	if err != nil {
		return err
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
		return err
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
	for i := 0; i < request.Cycles; i += 1 {
	}

	return nil
}

func (handler *ResourceRequestHandler) RunNetworkRequest(request *NetworkRequest) error {
	url := fmt.Sprintf("http://%s:7998/network-endpoint", handler.NetworkPeers.Value)
	handler.NetworkPeers = handler.NetworkPeers.Next()

	content := make([]byte, request.Bandwidth)
	resp, err := http.Post(url, "text/plain", bytes.NewReader(content))

	if err != nil {
		return err
	} else if resp.StatusCode != http.StatusOK {
		return errors.New(fmt.Sprintf("Return code %d", resp.StatusCode))
	}
	return nil
}

func (handler *ResourceRequestHandler) RunBlkIoRequest(request *BlkIoRequest) error {
	return nil
}

func (handler *ResourceRequestHandler) Run(request *ResourceRequest) error {
	if request.CPURequest != nil {
		return handler.RunCPURequest(request.CPURequest)
	} else if request.NetworkRequest != nil {
		return handler.RunNetworkRequest(request.NetworkRequest)
	} else if request.BlkIoRequest != nil {
		return handler.RunBlkIoRequest(request.BlkIoRequest)
	}
	return nil
}


type Work struct {
	Requests []ResourceRequest `form:"definitions" json:"definitions"`
}

func main() {

	prometheus.MustRegister(duration)
	prometheus.MustRegister(counter)

	handler := &ResourceRequestHandler{}
	handler.GetKubeClient()
	handler.GetNetworkPeers()

	r := gin.Default()

	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	r.POST("/network-endpoint", func(c *gin.Context) {
		body, err := ioutil.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error": err.Error(),
			})
		}
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"length": len(body),
		})
	})

	r.POST("/run", func(c *gin.Context) {

		defer func(begun time.Time) {
			duration.Observe(time.Since(begun).Seconds())
			counter.With(prometheus.Labels{
				"resource": "cpu",
			}).Inc()
		}(time.Now())

		startTime := time.Now()
		var work Work
		if err := c.BindJSON(&work); err == nil {
			for _, definition := range work.Requests {
				if err := handler.Run(&definition); err != nil {
					// Error
				}
			}

			// Run the worker
			c.JSON(http.StatusOK, gin.H{
				"message":  "done",
				"duration": time.Since(startTime).Seconds(),
				// Put the results / summaries of the work in response
			})
		} else {
			// Handle error
		}
	})

	r.Run(":7998")
}
