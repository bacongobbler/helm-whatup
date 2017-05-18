package main

import (
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/spf13/cobra"

	"k8s.io/helm/pkg/helm"
	"k8s.io/helm/pkg/proto/hapi/release"
	"k8s.io/helm/pkg/repo"
)

const globalUsage = `
Check to see if there is an updated version available for installed charts.
`

var version = "canary"

func main() {
	cmd := &cobra.Command{
		Use:   "whatup [flags]",
		Short: fmt.Sprintf("check if installed charts are out of date (helm-whatup %s)", version),
		RunE:  run,
	}

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) error {
	var (
		indices  []*repo.IndexFile
		releases []*release.Release
		wg       sync.WaitGroup
	)
	client := helm.NewClient(helm.Host(os.Getenv("TILLER_HOST")))

	wg.Add(2)

	go fetchReleases(client, releases, wg)
	go fetchIndices(client, indices, wg)

	wg.Wait()

	if len(releases) == 0 {
		log.Info("No releases found. All up to date!")
		return nil
	}

	if len(repositories) == 0 {
		log.Info("No repositories found. Did you run `helm repo update`?")
		return nil
	}

	for _, release := range releases {
		for _, idx := range indices {
			if idx.Has(release.Name, release.Version) {
				// fetch latest release
				chartVer, err := idx.Get(release.Name, "")
				if err != nil {
					log.Fatal(err)
				}
				if chartVer.Version != release.Version {
					// if it differs, then there's an updata available.
					log.Info(fmt.Sprintf("there is an update available for %s.\nInstalled version: %s\nAvailable version: %s", release.Name, release.Version, chartVer.Version))
				}
			}
		}
	}
}

func fetchReleases(client *helm.Client, releases []*release.Release, wg *sync.WaitGroup) {
	defer wg.Done()
	res, err := client.ListReleases(helm.ReleaseListStatuses([]release.Status_Code{release.Status_DEPLOYED}))
	if err != nil {
		log.Fatal(err)
	}
	releases = res.Releases
}

func fetchIndices(client *helm.Client, indices []*repo.IndexFile, wg *sync.WaitGroup) {
	defer wg.Done()
	rfp := os.Getenv("HELM_PATH_REPOSITORY_FILE")
	repofile, err := repo.LoadRepositoriesFile(rfp)
	if err != nil {
		log.Fatalf("could not load repositories file '%s': %s", rfp, err)
	}
	for _, repository := range repofile {
		idx, err := repo.LoadIndexFile(repository.Cache)
		if err != nil {
			log.Fatalf("could not load index file '%s': %s", repository.Cache, err)
		}
		indices := append(indices, idx)
	}
}
