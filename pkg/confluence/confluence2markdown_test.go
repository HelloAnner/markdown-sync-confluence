package confluence

import (
	"testing"

	"github.com/HelloAnner/markdown-sync-confluence/pkg/config"
)


func TestDownload(t *testing.T) {
	cliConfig := map[string]string{
	}
	cfg, _ := config.LoadConfig(cliConfig)
	converter := NewConverter(cfg)
	converter.SearchAndDownloadToLocalFile("Swift" , 500)
}
