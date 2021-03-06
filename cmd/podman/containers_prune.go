package main

import (
	"context"

	"github.com/containers/libpod/cmd/podman/shared"
	"github.com/containers/libpod/libpod"
	"github.com/containers/libpod/libpod/adapter"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var (
	pruneContainersDescription = `
	podman container prune

	Removes all exited containers
`

	pruneContainersCommand = cli.Command{
		Name:         "prune",
		Usage:        "Remove all stopped containers",
		Description:  pruneContainersDescription,
		Action:       pruneContainersCmd,
		OnUsageError: usageErrorHandler,
	}
)

func pruneContainers(runtime *adapter.LocalRuntime, ctx context.Context, maxWorkers int, force bool) error {
	var deleteFuncs []shared.ParallelWorkerInput

	filter := func(c *libpod.Container) bool {
		state, err := c.State()
		if state == libpod.ContainerStateStopped || (state == libpod.ContainerStateExited && err == nil && c.PodID() == "") {
			return true
		}
		return false
	}
	delContainers, err := runtime.GetContainers(filter)
	if err != nil {
		return err
	}
	if len(delContainers) < 1 {
		return nil
	}
	for _, container := range delContainers {
		con := container
		f := func() error {
			return runtime.RemoveContainer(ctx, con, force)
		}

		deleteFuncs = append(deleteFuncs, shared.ParallelWorkerInput{
			ContainerID:  con.ID(),
			ParallelFunc: f,
		})
	}
	// Run the parallel funcs
	deleteErrors, errCount := shared.ParallelExecuteWorkerPool(maxWorkers, deleteFuncs)
	return printParallelOutput(deleteErrors, errCount)
}

func pruneContainersCmd(c *cli.Context) error {
	runtime, err := adapter.GetRuntime(c)
	if err != nil {
		return errors.Wrapf(err, "could not get runtime")
	}
	defer runtime.Shutdown(false)

	maxWorkers := shared.Parallelize("rm")
	if c.GlobalIsSet("max-workers") {
		maxWorkers = c.GlobalInt("max-workers")
	}
	logrus.Debugf("Setting maximum workers to %d", maxWorkers)

	return pruneContainers(runtime, getContext(), maxWorkers, c.Bool("force"))
}
