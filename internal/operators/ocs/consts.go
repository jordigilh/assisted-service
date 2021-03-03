package ocs

type resource int8

const (
	memory resource = iota
	cpu
)

type ocsDeploymentType string

const (
	minimal  ocsDeploymentType = "Minimal Deployment Mode"
	compact  ocsDeploymentType = "Compact Mode"
	standard ocsDeploymentType = "Standard Deployment Mode"
	none     ocsDeploymentType = ""
)

const (
	//ssdDrive represents a Solid State Disk Drive
	ssdDrive string = "SSD"
	//hdDrive represents a Hard Disk Drive
	hdDrive string = "HDD"
)
