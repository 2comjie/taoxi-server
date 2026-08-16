package gen

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"text/template"

	"github.com/2comjie/wali/core/util"
)

const runTemplate = `<component name="ProjectRunConfigurationManager">
  <configuration default="false" name="{{.Name}}" type="GoApplicationRunConfiguration" factoryName="Go Application">
    <module name="{{.ModuleName}}" />
    <working_directory value="$PROJECT_DIR${{.WorkDir}}" />
    <useCustomBuildTags value="true" />
    <parameters value="{{.Args}}" />
    <kind value="PACKAGE" />
    <package value="{{.Package}}" />
    <directory value="$PROJECT_DIR$" />
    <method v="2" />
  </configuration>
</component>`

type RunConfig struct {
	Name       string
	ModuleName string
	Args       string
	Package    string
	WorkDir    string
}

func TestGenerate(t *testing.T) {
	moduleRoot, err := util.GetModuleRootDir()
	if err != nil {
		t.Fatal(err)
	}

	projectRoot := moduleRoot
	workDir := ""
	workspaceRoot := filepath.Dir(moduleRoot)
	if _, err = os.Stat(filepath.Join(workspaceRoot, "go.work")); err == nil {
		projectRoot = workspaceRoot
		workDir = "/" + filepath.Base(moduleRoot)
	}

	tpl, err := template.New("run").Parse(runTemplate)
	if err != nil {
		t.Fatal(err)
	}

	savePath := filepath.Join(projectRoot, ".run")
	if err = os.MkdirAll(savePath, 0o755); err != nil {
		t.Fatal(err)
	}

	moduleName := filepath.Base(projectRoot)
	configs := []RunConfig{
		{
			Name:       "01_Api_1",
			ModuleName: moduleName,
			Args:       "-service-name api -service-index 1 -env Local",
			Package:    "github.com/2comjie/taoxi-server/app/Api",
			WorkDir:    workDir,
		},
		{
			Name:       "02_Gate_1",
			ModuleName: moduleName,
			Args:       "-service-name gate -service-index 1 -env Local",
			Package:    "github.com/2comjie/taoxi-server/app/Gate",
			WorkDir:    workDir,
		},
		{
			Name:       "03_Gate_2",
			ModuleName: moduleName,
			Args:       "-service-name gate -service-index 2 -env Local",
			Package:    "github.com/2comjie/taoxi-server/app/Gate",
			WorkDir:    workDir,
		},
		{
			Name:       "04_Player_1",
			ModuleName: moduleName,
			Args:       "-service-name player -service-index 1 -env Local",
			Package:    "github.com/2comjie/taoxi-server/app/Player",
			WorkDir:    workDir,
		},
		{
			Name:       "05_Player_2",
			ModuleName: moduleName,
			Args:       "-service-name player -service-index 2 -env Local",
			Package:    "github.com/2comjie/taoxi-server/app/Player",
			WorkDir:    workDir,
		},
	}

	for _, config := range configs {
		var buffer bytes.Buffer
		if err = tpl.Execute(&buffer, config); err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(savePath, config.Name+".run.xml")
		if err = os.WriteFile(path, buffer.Bytes(), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}
