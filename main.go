package main

import (
	"fmt"
	"os"

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
	client := helm.NewClient(helm.Host(os.Getenv("TILLER_HOST")))

	releases, err := fetchReleases(client)
	if err != nil {
		return err
	}

	repositories, err := fetchIndices(client)
	if err != nil {
		return err
	}

	if len(releases) == 0 {
		fmt.Println("No releases found. All up to date!")
		return nil
	}

	if len(repositories) == 0 {
		fmt.Println("No repositories found. Did you run `helm repo update`?")
		return nil
	}

	for _, release := range releases {
		for _, idx := range repositories {
			if idx.Has(release.Chart.Metadata.Name, release.Chart.Metadata.Version) {
				// fetch latest release
				chartVer, err := idx.Get(release.Chart.Metadata.Name, "")
				if err != nil {
					return err
				}
				if chartVer.Version != release.Chart.Metadata.Version {
					// if it differs, then there's an update available.
					fmt.Printf("There is an update available for release %s (%s)!\nInstalled version: %s\nAvailable version: %s\n", release.Name, release.Chart.Metadata.Name, release.Chart.Metadata.Version, chartVer.Version)
				} else {
					fmt.Printf("Release %s (%s) is up to date (%s).\n", release.Name, release.Chart.Metadata.Name, release.Chart.Metadata.Version)
				}
			}
		}
	}
	fmt.Println("Done.")
	return nil
}

func fetchReleases(client *helm.Client) ([]*release.Release, error) {
	res, err := client.ListReleases()
	if err != nil {
		return nil, err
	}
	return res.Releases, nil
}

func fetchIndices(client *helm.Client) ([]*repo.IndexFile, error) {
	indices := []*repo.IndexFile{}
	rfp := os.Getenv("HELM_PATH_REPOSITORY_FILE")
	repofile, err := repo.LoadRepositoriesFile(rfp)
	if err != nil {
		return nil, fmt.Errorf("could not load repositories file '%s': %s", rfp, err)
	}
	for _, repository := range repofile.Repositories {
		idx, err := repo.LoadIndexFile(repository.Cache)
		if err != nil {
			return nil, fmt.Errorf("could not load index file '%s': %s", repository.Cache, err)
		}
		indices = append(indices, idx)
	}
	return indices, nil
}
