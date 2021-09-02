package config

// *************************************
// THIS FILE IS GENERATED - DO NOT EDIT
// *************************************

import "github.com/atlasmap/atlasmap-operator/pkg/util"

// AtlasMapConfig --
type AtlasMapConfig struct {
	AtlasMapImage string
	Version       string
}

// DefaultConfiguration --
var DefaultConfiguration = AtlasMapConfig{
	AtlasMapImage: "docker.io/atlasmap/atlasmap",
	Version:       "latest",
}

func (c *AtlasMapConfig) GetAtlasMapImage() string {
	return util.ImageName(c.AtlasMapImage, c.Version)
}
