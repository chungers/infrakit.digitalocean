package instance

import (
	"context"
	"fmt"
	"testing"

	"github.com/digitalocean/godo"
	itypes "github.com/docker/infrakit.digitalocean/plugin/instance/types"
	"github.com/docker/infrakit/pkg/spi/instance"
	"github.com/docker/infrakit/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLabels(t *testing.T) {
	plugin := &plugin{
		tags: &fakeTagsService{},
	}
	id := instance.ID("foo")
	err := plugin.Label(id, map[string]string{
		"foo":    "bar",
		"banana": "baz",
	})

	require.NoError(t, err)
}

func TestLabelFails(t *testing.T) {
	plugin := &plugin{
		tags: &fakeTagsService{
			expectedErr: "something went wrong",
		},
	}
	id := instance.ID("foo")
	err := plugin.Label(id, map[string]string{
		"foo": "bar",
	})

	require.Error(t, err)
}

func TestValidate(t *testing.T) {
	plugin := &plugin{}
	err := plugin.Validate(types.AnyString(`{"Size":"1gb", "Image": "debian-8-x64"}`))

	require.NoError(t, err)
}

func TestValidateFails(t *testing.T) {
	plugin := &plugin{}
	err := plugin.Validate(types.AnyString("-"))

	require.Error(t, err)
}

func TestDestroyFails(t *testing.T) {
	plugin := &plugin{
		droplets: &fakeDropletsServices{
			expectedErr: "something went wrong",
		},
	}
	id := instance.ID("foo")
	err := plugin.Destroy(id)

	require.EqualError(t, err, "strconv.Atoi: parsing \"foo\": invalid syntax")

	id = instance.ID("12345")
	err = plugin.Destroy(id)

	require.EqualError(t, err, "something went wrong")
}

func TestDestroy(t *testing.T) {
	// FIXME(vdemeester) make a better test :D
	plugin := &plugin{
		droplets: &fakeDropletsServices{},
	}
	id := instance.ID("12345")
	err := plugin.Destroy(id)

	require.NoError(t, err)
}

func TestProvisionFailsInvalidProperties(t *testing.T) {
	spec := instance.Spec{
		Properties: types.AnyString(`{
  "NamePrefix": "bar",
  "tags": {
    "foo": "bar",
  }
}`),
	}
	plugin := &plugin{
		droplets: &fakeDropletsServices{},
	}
	_, err := plugin.Provision(spec)
	require.Error(t, err)
}

func TestProvisionFails(t *testing.T) {
	spec := instance.Spec{
		Properties: types.AnyString(`{
  "NamePrefix": "foo",
  "Size": "512mb",
  "Image": "ubuntu-14-04-x64",
  "Tags": ["foo"]
}`),
	}
	region := "asm2"
	plugin := &plugin{
		region: region,
		droplets: &fakeDropletsServices{
			expectedErr: "something went wrong",
		},
	}
	_, err := plugin.Provision(spec)
	require.EqualError(t, err, "something went wrong")
}

func TestProvision(t *testing.T) {
	spec := instance.Spec{
		Properties: types.AnyString(`{
  "NamePrefix": "foo",
  "Size": "512mb",
  "Image": "ubuntu-14-04-x64",
  "Tags": ["foo"]
}`),
	}
	region := "asm2"
	versiontag := fmt.Sprintf("%s:%s", itypes.InfrakitDOVersion, itypes.InfrakitDOCurrentVersion)
	plugin := &plugin{
		region: region,
		droplets: &fakeDropletsServices{
			createfunc: func(ctx context.Context, req *godo.DropletCreateRequest) (*godo.Droplet, *godo.Response, error) {
				assert.Contains(t, req.Name, "foo")
				assert.Equal(t, region, req.Region)
				assert.Equal(t, "512mb", req.Size)
				assert.Equal(t, godo.DropletCreateImage{
					Slug: "ubuntu-14-04-x64",
				}, req.Image)
				assert.Condition(t, isInSlice("foo", req.Tags))
				assert.Condition(t, isInSlice(versiontag, req.Tags))
				return &godo.Droplet{
					ID: 12345,
				}, nil, nil
			},
		},
	}
	id, err := plugin.Provision(spec)
	require.NoError(t, err)
	expectedID := instance.ID("12345")
	assert.Equal(t, &expectedID, id)
}

func isInSlice(s string, strings []string) assert.Comparison {
	return func() bool {
		isIn := false
		for _, str := range strings {
			if s == str {
				isIn = true
			}
		}
		return isIn
	}
}

