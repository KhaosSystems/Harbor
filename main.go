package main

import (
	"embed"
	"log"
	"runtime"

	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed all:frontend/dist
var assets embed.FS

func init() {
	application.RegisterEvent[string]("harbor:repositories-updated")
}

func main() {
	gitService := &GitService{}

	app := application.New(application.Options{
		Name:        "harbor",
		Description: "Harbor Git Client",
		Services: []application.Service{
			application.NewService(gitService),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	mainMenu := app.NewMenu()
	if runtime.GOOS == "darwin" {
		mainMenu.Append(application.NewMenuFromItems(application.NewAppMenu()))
	}
	fileMenu := mainMenu.AddSubmenu("File")
	fileMenu.Add("Add Local Repository...").SetAccelerator("CmdOrCtrl+O").OnClick(func(_ *application.Context) {
		result := gitService.SelectAndAddLocalRepository()
		if !result.Success {
			app.Logger.Error(result.Error)
		}
	})
	fileMenu.AddSeparator()
	fileMenu.Add("Quit").SetAccelerator("CmdOrCtrl+Q").OnClick(func(_ *application.Context) {
		app.Quit()
	})
	app.Menu.SetApplicationMenu(mainMenu)

	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title: "Harbor",
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 50,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInset,
		},
		BackgroundColour: application.NewRGB(27, 38, 54),
		URL:              "/",
	})

	err := app.Run()
	if err != nil {
		log.Fatal(err)
	}
}
