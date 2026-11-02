package pages

import (
	"context"
	"fmt"

	"Wavelet/openflare/plugins/server/kernel/repository"
)

func CleanupRetainedVersions(ctx context.Context) error {
	projects, err := repository.ListPagesProjects(ctx)
	if err != nil {
		return fmt.Errorf("list Pages projects for retention: %w", err)
	}
	for _, project := range projects {
		if err := pruneProjectDeploymentHistory(ctx, project.ID, 5, 0); err != nil {
			return fmt.Errorf("cleanup Pages project %d deployment history: %w", project.ID, err)
		}
	}
	return nil
}
