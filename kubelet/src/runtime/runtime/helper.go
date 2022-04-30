package runtime

import (
	"fmt"
	"github.com/docker/go-connections/nat"
	"minik8s/apiObject"
	"minik8s/kubelet/src/podutil"
	"minik8s/kubelet/src/runtime/container"
	"minik8s/kubelet/src/runtime/image"
	"minik8s/kubelet/src/types"
	"time"
)

// needPullImage judges whether we need pull the image of given container spec
func (rm *runtimeManager) needPullImage(container *apiObject.Container) (bool, error) {
	if container.ImagePullPolicy == apiObject.PullPolicyAlways {
		return true, nil
	}
	exist, err := rm.im.ExistsImage(container.Image)
	return !exist, err
}

// toFormattedEnv changes containerEnv to adapted form, like "FOO=bar"
// where FOO is name and bar is value
func (rm *runtimeManager) toFormattedEnv(containerEnv []apiObject.EnvVar) []string {
	var env []string
	for _, ev := range containerEnv {
		env = append(env, ev.Name+"="+ev.Value)
	}
	return env
}

// toVolumeBinds returns the binds of volumes
func (rm *runtimeManager) toVolumeBinds() []string {
	/// TODO implement it
	return nil
}

func (rm *runtimeManager) pauseContainerFullName(podFullName string, podUID types.UID) string {
	return podutil.ContainerFullName(pauseContainerName, podFullName, podUID, 0)
}

func (rm *runtimeManager) toPauseContainerReference(podFullName string, podUID types.UID) string {
	return podutil.ToContainerReference(rm.pauseContainerFullName(podFullName, podUID))
}

func (rm *runtimeManager) addPortBindings(portBindings container.PortBindings, ports []apiObject.ContainerPort) error {
	for _, port := range ports {
		if port.Protocol == "" {
			port.Protocol = "tcp"
		}
		containerPort, err := nat.NewPort(port.Protocol, port.ContainerPort)
		if err != nil {
			return err
		}
		if port.HostIP == "" {
			port.HostIP = "127.0.0.1"
		}
		portBindings[containerPort] = []nat.PortBinding{{
			HostIP:   port.HostIP,
			HostPort: port.HostPort,
		}}
	}
	return nil
}

func (rm *runtimeManager) addPortSet(portSet container.PortSet, ports []apiObject.ContainerPort) {
	for _, port := range ports {
		portSet[container.Port(port.ContainerPort+"/tcp")] = struct{}{}
	}
}

func (rm *runtimeManager) getPauseContainerCreateConfig(pod *apiObject.Pod) (*container.ContainerCreateConfig, error) {
	labels := map[string]string{
		KubernetesPodUIDLabel: pod.UID(),
	}

	portBindings := container.PortBindings{}
	portSet := container.PortSet{}
	for _, c := range pod.Spec.Containers {
		err := rm.addPortBindings(portBindings, c.Ports)
		if err != nil {
			return nil, err
		}
		rm.addPortSet(portSet, c.Ports)
	}

	return &container.ContainerCreateConfig{
		Image:        pauseImage,
		Volumes:      nil,
		Labels:       labels,
		Binds:        nil,
		IpcMode:      "shareable",
		ExposedPorts: portSet,
		PortBindings: portBindings,
	}, nil
}

func (rm *runtimeManager) getCommonContainerCreateConfig(c *apiObject.Container, podFullName string, podUID types.UID) *container.ContainerCreateConfig {
	// the label of given podUID
	labels := map[string]string{
		KubernetesPodUIDLabel: podUID,
	}
	pauseContainerFullName := rm.pauseContainerFullName(podFullName, podUID)
	pauseContainerRef := rm.toPauseContainerReference(podFullName, podUID)
	return &container.ContainerCreateConfig{
		Image:       c.Image,
		Entrypoint:  c.Command,
		Cmd:         c.Args,
		Env:         rm.toFormattedEnv(c.Env),
		Volumes:     nil,
		Labels:      labels,
		Tty:         c.TTY,
		NetworkMode: container.NetworkMode(pauseContainerRef),
		IpcMode:     container.IpcMode(pauseContainerRef),
		PidMode:     container.PidMode(pauseContainerRef),
		Binds:       nil,
		VolumesFrom: []string{pauseContainerFullName},
	}
}

func (rm *runtimeManager) inspectionToContainerStatus(inspection *container.ContainerInspectInfo) (*container.ContainerStatus, error) {
	state := container.ContainerStateUnknown
	switch inspection.State.Status {
	case "running":
		state = container.ContainerStateRunning
	case "created":
		state = container.ContainerStateCreated
	case "exited":
		state = container.ContainerStateExited
	}

	createdAt, err := time.Parse(time.RFC3339Nano, inspection.Created)
	if err != nil {
		return nil, err
	}

	startedAt, err := time.Parse(time.RFC3339Nano, inspection.State.StartedAt)
	if err != nil {
		return nil, err
	}

	finishedAt, err := time.Parse(time.RFC3339Nano, inspection.State.FinishedAt)
	if err != nil {
		return nil, err
	}

	return &container.ContainerStatus{
		ID:           inspection.ID,
		Name:         inspection.Name,
		State:        state,
		CreatedAt:    createdAt,
		StartedAt:    startedAt,
		FinishedAt:   finishedAt,
		ExitCode:     inspection.State.ExitCode,
		ImageID:      inspection.Image,
		RestartCount: inspection.RestartCount,
		Error:        inspection.State.Error,
	}, nil
}

func (rm *runtimeManager) getPodContainerStatuses(pod *apiObject.Pod) ([]*container.ContainerStatus, error) {
	containers, err := rm.cm.ListContainers(&container.ContainerListConfig{
		All: true,
		LabelSelector: container.LabelSelector{
			KubernetesPodUIDLabel: pod.UID(),
		},
	})
	if err != nil {
		return nil, err
	}

	containerStatuses := make([]*container.ContainerStatus, len(containers))
	for i, c := range containers {
		inspection, err := rm.cm.InspectContainer(c.ID)
		if err != nil {
			return nil, err
		}
		containerStatuses[i], err = rm.inspectionToContainerStatus(&inspection)
		if err != nil {
			return nil, err
		}
	}
	return containerStatuses, nil
}

// startPauseContainer starts the pause container that other common containers need
func (rm *runtimeManager) startPauseContainer(pod *apiObject.Pod) error {
	// Step 1: Do we need pull the image?
	exists, err := rm.im.ExistsImage(pauseImage)
	if err != nil {
		return err
	}

	// Step 2: If needed, pull the image for the given container
	if !exists {
		fmt.Println("Need to pull image", pauseImage)
		err = rm.im.PullImage(pauseImage, &image.ImagePullConfig{
			Verbose: true,
			All:     false,
		})
		if err != nil {
			return err
		}
	} else {
		fmt.Printf("No need to pull image %s, continue\n", pauseImage)
	}

	// Prepare
	podFullName := pod.FullName()
	podUID := pod.UID()

	// Step 3: Create a container
	fmt.Println("Now create the container")

	containerFullName := rm.pauseContainerFullName(podFullName, podUID)

	// get the container create config of pause
	var createConfig *container.ContainerCreateConfig
	createConfig, err = rm.getPauseContainerCreateConfig(pod)
	if err != nil {
		return err
	}

	var ID container.ContainerID
	ID, err = rm.cm.CreateContainer(containerFullName, createConfig)
	if err != nil {
		return err
	}
	fmt.Println("Create the container successfully, got ID", ID)

	// Step 4: Start this container
	fmt.Println("Now start the container with ID", ID)
	err = rm.cm.StartContainer(ID, &container.ContainerStartConfig{})
	return err
}

// startPauseContainer starts the pause container that other common containers need
func (rm *runtimeManager) removePauseContainer(pod *apiObject.Pod) error {
	// Prepare
	podFullName := pod.FullName()
	podUID := pod.UID()

	containerFullName := rm.pauseContainerFullName(podFullName, podUID)
	err := rm.cm.StopContainer(containerFullName, &container.ContainerStopConfig{})
	if err != nil {
		return err
	}
	return rm.cm.RemoveContainer(containerFullName, &container.ContainerRemoveConfig{})
}

func (rm *runtimeManager) removePodCommonContainers(pod *apiObject.Pod) error {
	// Prepare
	containers, err := rm.cm.ListContainers(&container.ContainerListConfig{
		All: true,
		LabelSelector: container.LabelSelector{
			KubernetesPodUIDLabel: pod.UID(),
		}},
	)

	if err != nil {
		return err
	}

	pauseContainerFullName := "/" + rm.pauseContainerFullName(pod.FullName(), pod.UID())

	for _, c := range containers {
		// Not include pause container
		if c.Name == pauseContainerFullName {
			continue
		}
		err = rm.cm.StopContainer(c.ID, &container.ContainerStopConfig{})
		if err != nil {
			return err
		}

		err = rm.cm.RemoveContainer(c.ID, &container.ContainerRemoveConfig{})
		if err != nil {
			return err
		}
	}
	return nil
}

// startCommonContainer starts a common container according to the given spec
func (rm *runtimeManager) startCommonContainer(pod *apiObject.Pod, c *apiObject.Container) error {
	// Step 1: Do we need pull the image?
	needPull, err := rm.needPullImage(c)
	if err != nil {
		return err
	}

	// Step 2: If needed, pull the image for the given container
	if needPull {
		fmt.Println("Need to pull image", c.Image)
		err = rm.im.PullImage(c.Image, &image.ImagePullConfig{
			Verbose: true,
			All:     false,
		})
		if err != nil {
			return err
		}
	} else {
		fmt.Printf("No need to pull image %s, continue\n", c.Image)
	}

	// Prepare
	podFullName := pod.FullName()
	podUID := pod.UID()

	// Step 3: Create a container
	fmt.Println("Now create the container")

	containerFullName := podutil.ContainerFullName(c.Name, podFullName, podUID, 0)

	var ID container.ContainerID
	ID, err = rm.cm.CreateContainer(containerFullName, rm.getCommonContainerCreateConfig(c, podFullName, podUID))
	if err != nil {
		return err
	}
	fmt.Println("Create the container successfully, got ID", ID)

	// Step 4: Start this container
	fmt.Println("Now start the container with ID", ID)
	err = rm.cm.StartContainer(ID, &container.ContainerStartConfig{})
	return err
}

func (rm *runtimeManager) getAllPodContainers() (map[types.UID][]*container.ContainerStatus, error) {
	containers, err := rm.cm.ListContainers(&container.ContainerListConfig{
		All: true,
		LabelSelector: container.LabelSelector{
			KubernetesPodUIDLabel: "",
		}})
	if err != nil {
		return nil, err
	}

	containerStatuses := make(map[types.UID][]*container.ContainerStatus)
	for _, c := range containers {
		inspection, err := rm.cm.InspectContainer(c.ID)
		if err != nil {
			return nil, err
		}
		var cs *container.ContainerStatus
		if podUID, exists := inspection.Config.Labels[KubernetesPodUIDLabel]; exists {
			cs, err = rm.inspectionToContainerStatus(&inspection)
			if err != nil {
				return nil, err
			}
			fmt.Printf("Container %s belongs to pod %s\n", cs.Name, podUID)
			containerStatuses[podUID] = append(containerStatuses[podUID], cs)
		} else {
			panic("It's impossible!")
		}
	}
	return containerStatuses, nil
}
