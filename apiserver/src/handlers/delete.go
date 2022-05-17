package handlers

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"minik8s/apiObject"
	"minik8s/apiserver/src/etcd"
	"minik8s/apiserver/src/url"
	"minik8s/entity"
	"minik8s/listwatch"
	"minik8s/util/topicutil"
	"net/http"
	"path"
)

func deleteSpecifiedNode(namespace, name string) (err error) {
	log("Node to delete is %s/%s", namespace, name)
	etcdNodeURL := path.Join(url.NodeURL, namespace, name)
	if err = etcd.Delete(etcdNodeURL); err == nil {
		etcdNodeStatusURL := path.Join(url.NodeURL, "status", namespace, name)
		err = etcd.Delete(etcdNodeStatusURL)
	}
	return
}

func deleteSpecifiedPod(namespace, name string) (pod *apiObject.Pod, err error) {
	log("Pod to delete is %s/%s", namespace, name)

	etcdPodStatusURL := path.Join(url.PodURL, "status", namespace, name)
	_ = etcd.Delete(etcdPodStatusURL)

	var raw string
	etcdPodURL := path.Join(url.PodURL, namespace, name)
	if raw, err = etcd.Get(etcdPodURL); err != nil {
		return nil, err
	}

	if err = json.Unmarshal([]byte(raw), &pod); err != nil {
		return nil, fmt.Errorf("no such pod %s/%s", namespace, name)
	}

	err = etcd.Delete(etcdPodURL)
	return
}

func deleteSpecifiedReplicaSet(namespace, name string) (rs *apiObject.ReplicaSet, err error) {
	log("Rs to delete is %s/%s", namespace, name)

	etcdReplicaSetStatusURL := path.Join(url.ReplicaSetURL, "status", namespace, name)
	_ = etcd.Delete(etcdReplicaSetStatusURL)

	var raw string
	etcdReplicaSetURL := path.Join(url.ReplicaSetURL, namespace, name)
	if raw, err = etcd.Get(etcdReplicaSetURL); err != nil {
		return nil, err
	}

	if err = json.Unmarshal([]byte(raw), &rs); err != nil {
		return nil, fmt.Errorf("no such replicaSet %s/%s", namespace, name)
	}

	err = etcd.Delete(etcdReplicaSetURL)
	return
}

func deleteSpecifiedHPA(namespace, name string) (hpa *apiObject.HorizontalPodAutoscaler, err error) {
	log("hpa to delete is %s/%s", namespace, name)

	etcdHPAStatusURL := path.Join(url.HPAURL, "status", namespace, name)
	_ = etcd.Delete(etcdHPAStatusURL)

	var raw string
	etcdHPAURL := path.Join(url.HPAURL, namespace, name)
	if raw, err = etcd.Get(etcdHPAURL); err != nil {
		return nil, err
	}

	if err = json.Unmarshal([]byte(raw), &hpa); err != nil {
		return nil, fmt.Errorf("no such hpa %s/%s", namespace, name)
	}

	err = etcd.Delete(etcdHPAURL)
	return
}

func HandleDeleteNode(c *gin.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")
	if err := deleteSpecifiedNode(namespace, name); err != nil {
		c.String(http.StatusOK, err.Error())
	}
	c.String(http.StatusOK, "Delete successfully")
}

func HandleDeletePod(c *gin.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")

	if podToDelete, err := deleteSpecifiedPod(namespace, name); err != nil {
		c.String(http.StatusOK, err.Error())
		return
	} else {
		podDeleteMsg, _ := json.Marshal(entity.PodUpdate{
			Action: entity.DeleteAction,
			Target: *podToDelete,
		})
		listwatch.Publish(topicutil.SchedulerPodUpdateTopic(), podDeleteMsg)
	}
	c.String(http.StatusOK, "Delete successfully")
}

func HandleDeleteReplicaSet(c *gin.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")

	if replicaSetToDelete, err := deleteSpecifiedReplicaSet(namespace, name); err != nil {
		c.String(http.StatusOK, err.Error())
		return
	} else {
		replicaSetDeleteMsg, _ := json.Marshal(entity.ReplicaSetUpdate{
			Action: entity.DeleteAction,
			Target: *replicaSetToDelete,
		})
		listwatch.Publish(topicutil.ReplicaSetUpdateTopic(), replicaSetDeleteMsg)
	}
	c.String(http.StatusOK, "Delete successfully")
}

func HandleDeleteHPA(c *gin.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")

	if hpaToDelete, err := deleteSpecifiedHPA(namespace, name); err != nil {
		c.String(http.StatusOK, err.Error())
		return
	} else {
		hpaDeleteMsg, _ := json.Marshal(entity.HPAUpdate{
			Action: entity.DeleteAction,
			Target: *hpaToDelete,
		})
		listwatch.Publish(topicutil.HPAUpdateTopic(), hpaDeleteMsg)
	}
	c.String(http.StatusOK, "Delete successfully")
}

func HandleReset(c *gin.Context) {
	if err := etcd.DeleteAllKeys(); err != nil {
		c.String(http.StatusOK, err.Error())
		return
	}
	c.String(http.StatusOK, "OK")
}
