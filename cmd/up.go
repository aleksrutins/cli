package cmd

import (
	"context"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/railwayapp/cli/entity"
	"github.com/railwayapp/cli/lib/git"
	CLIErrors "github.com/railwayapp/cli/errors"
	"github.com/railwayapp/cli/ui"
)

func (h *Handler) Up(ctx context.Context, req *entity.CommandRequest) error {
	isVerbose, err := req.Cmd.Flags().GetBool("verbose")
	if err != nil {
		// Verbose mode isn't a necessary flag; just default to false.
		isVerbose = false
	}

	serviceName, err := req.Cmd.Flags().GetString("service")
	if err != nil {
		return err
	}

	fmt.Print(ui.VerboseInfo(isVerbose, "Using verbose mode"))

	projectConfig, err := h.ctrl.GetProjectConfigs(ctx)
	if err != nil {
		return err
	}

	src := projectConfig.ProjectPath
	if src == "" {
		// When deploying with a project token, the project path is empty
		src = "."
	}

	fmt.Print(ui.VerboseInfo(isVerbose, fmt.Sprintf("Uploading directory %s", src)))

	fmt.Print(ui.VerboseInfo(isVerbose, "Loading environment"))
	environmentName, err := req.Cmd.Flags().GetString("environment")
	if err != nil {
		return err
	}

	environment, err := h.getEnvironment(ctx, environmentName)
	if err != nil {
		return err
	}
	fmt.Print(ui.VerboseInfo(isVerbose, fmt.Sprintf("Using environment %s", ui.Bold(environment.Name))))

	fmt.Print(ui.VerboseInfo(isVerbose, "Loading project"))
	project, err := h.ctrl.GetProject(ctx, projectConfig.Project)
	if err != nil {
		return err
	}

	serviceId := ""
	if serviceName != "" {
		for _, service := range project.Services {
			if service.Name == serviceName {
				serviceId = service.ID
			}
		}

		if serviceId == "" {
			return CLIErrors.ServiceNotFound
		}
	}

	// If service has not been provided via flag, prompt for it
	if serviceId == "" {
		fmt.Print(ui.VerboseInfo(isVerbose, "Loading services"))
		service, err := ui.PromptServices(project.Services)
		if err != nil {
			return err
		}

		if service != nil {
			serviceId = service.ID
		}
	}

	_, err = ioutil.ReadFile(".railwayignore")
	if err == nil {
		fmt.Print(ui.VerboseInfo(isVerbose, "Using ignore file .railwayignore"))
	}

	fmt.Print(ui.VerboseInfo(isVerbose, "Getting Git information"))
	gitInfo, err := git.GetAllMetadata(src)

	if !gitInfo.IsRepo || err != nil {
		fmt.Print(ui.VerboseInfo(isVerbose, "No Git repository found"))
	}
	fmt.Print(ui.VerboseInfo(isVerbose, fmt.Sprintf(`Git information:
	Repository name: %s
	Branch: %s
	Latest commit:
		Hash: %s
		Message: %s
		Author: %s
	`, gitInfo.RepoName, gitInfo.Branch, gitInfo.Commit.Hash, gitInfo.Commit.Message, gitInfo.Commit.Author)))

	ui.StartSpinner(&ui.SpinnerCfg{
		Message: "Laying tracks in the clouds...",
	})
	res, err := h.ctrl.Upload(ctx, &entity.UploadRequest{
		ProjectID:     projectConfig.Project,
		EnvironmentID: environment.Id,
		ServiceID:     serviceId,
		RootDir:       src,
		GitInfo:       gitInfo,
	})
	if err != nil {
		ui.StopSpinner("")
		return err
	} else {
		ui.StopSpinner(fmt.Sprintf("☁️ Build logs available at %s\n", ui.GrayText(res.URL)))
	}
	detach, err := req.Cmd.Flags().GetBool("detach")
	if err != nil {
		return err
	}
	if detach {
		return nil
	}

	for i := 0; i < 3; i++ {
		err = h.ctrl.GetActiveBuildLogs(ctx, 0)
		if err == nil {
			break
		}
		time.Sleep(time.Duration(i) * 250 * time.Millisecond)
	}

	fmt.Printf("\n\n======= Build Completed ======\n\n")

	err = h.ctrl.GetActiveDeploymentLogs(ctx, 1000)
	if err != nil {
		return err
	}

	fmt.Printf("☁️ Deployment logs available at %s\n", ui.GrayText(res.URL))
	fmt.Printf("OR run `railway logs` to tail them here\n\n")

	if res.DeploymentDomain != "" {
		fmt.Printf("☁️ Deployment live at %s\n", ui.GrayText(h.ctrl.GetFullUrlFromStaticUrl(res.DeploymentDomain)))
	} else {
		fmt.Printf("☁️ Deployment is live\n")
	}

	return nil
}
