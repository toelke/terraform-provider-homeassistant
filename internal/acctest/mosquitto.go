package acctest

import (
	"context"
	"testing"
	"time"

	docker "github.com/moby/moby/api/types/container"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// MosquittoImage is the MQTT broker for the MQTT acceptance tests.
const MosquittoImage = "eclipse-mosquitto:2"

// StartMosquitto starts an MQTT broker in the network namespace of the shared Home Assistant, so
// Home Assistant reaches it at localhost:1883. Without a config file, mosquitto listens only on
// localhost and allows anonymous clients. The broker is removed when the test ends.
func StartMosquitto(t *testing.T) {
	t.Helper()
	SharedInstance(t)
	ctx := context.Background()
	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image: MosquittoImage,
			HostConfigModifier: func(hc *docker.HostConfig) {
				hc.NetworkMode = docker.NetworkMode("container:" + container.GetContainerID())
			},
			WaitingFor: wait.ForLog("running").WithStartupTimeout(time.Minute),
		},
		Started: true,
	})
	if c != nil {
		t.Cleanup(func() {
			if err := c.Terminate(context.Background(), testcontainers.StopTimeout(time.Second)); err != nil {
				t.Logf("terminate mosquitto: %v", err)
			}
		})
	}
	if err != nil {
		t.Fatalf("start mosquitto: %v", err)
	}
}
