package container

import (
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"time"
)

type ContainerCreateConfig struct {
	Image        string              // Name of the image as it was passed by the operator (e.g. could be symbolic)
	Entrypoint   CmdLine             // Entrypoint to run when starting the container
	Cmd          CmdLine             // Command to run when starting the container
	Env          []string            // List of environment variable to set in the container
	Volumes      map[string]struct{} // List of volumes (mounts) used for the container
	Labels       map[string]string   // List of labels set to this container
	IpcMode      IpcMode             // IPC namespace to use for the container
	PidMode      PidMode             // PID namespace to use for the container
	ExposedPorts nat.PortSet         `json:",omitempty"` // List of exposed ports
	Tty          bool                // Attach standard streams to a tty, including stdin if it is not closed.
	Links        []string            // List of links (in the name:alias form)
	NetworkMode  NetworkMode         // Network mode to use for the container, e.g., --network=container:nginx
	Binds        []string            // List of volume bindings for this container
	PortBindings PortBindings        // Port mapping between the exposed port (container) and the host
	VolumesFrom  []string            // List of volumes to take from other container
}

type IpcMode = container.IpcMode

type PidMode = container.PidMode

type NetworkMode = container.NetworkMode

type Port = nat.Port

type PortSet = nat.PortSet

type PortBindings = nat.PortMap

type RemoveConfig = types.ContainerRemoveOptions

type LabelSelector = map[string]string

type ListConfig struct {
	Quiet         bool
	Size          bool
	All           bool
	Latest        bool
	Since         string
	Before        string
	Limit         int
	LabelSelector LabelSelector
}

type StartConfig = types.ContainerStartOptions

type InspectInfo = types.ContainerJSON

type StopConfig struct {
	timeout time.Duration
}
