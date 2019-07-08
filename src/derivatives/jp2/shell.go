// shell.go contains all the ugly shell commands we need to be able to execute

package jp2

import (
	"fmt"

	"github.com/uoregon-libraries/newspaper-curation-app/src/shell"
)

func (t *Transformer) makePNGFromPDF() bool {
	return shell.ExecSubgroup(t.GhostScript, t.Logger, "-dNumRenderingThreads=4", "-dNOPAUSE",
		"-sDEVICE=png16m", "-dFirstPage=1", "-dLastPage=1",
		"-dBackgroundColor=16#ffffff", "-sOutputFile="+t.tmpPNG,
		fmt.Sprintf("-r%d", t.PDFResolution), "-q", t.SourceFile, "-c", "quit")
}

func (t *Transformer) makePNGFromTIFF() bool {
	return shell.ExecSubgroup(t.GraphicsMagick, t.Logger, "convert", "-background", "white",
		"-quality", "0", t.SourceFile, t.tmpPNG)
}

func (t *Transformer) makeJP2FromPNG(rate float64) bool {
	return shell.ExecSubgroup(t.OPJCompress, t.Logger, "-i", t.tmpPNG, "-o", t.tmpJP2, "-t",
		"1024,1024", "-r", fmt.Sprintf("%0.3f", rate))
}

func (t *Transformer) makeJP2FromPNGDashI(rate float64) bool {
	return shell.ExecSubgroup(t.OPJCompress, t.Logger, "-i", t.tmpPNG, "-o", t.tmpJP2, "-t",
		"1024,1024", "-r", fmt.Sprintf("%0.3f", rate), "-I")
}

func (t *Transformer) testJP2Decompress() bool {
	return shell.ExecSubgroup(t.OPJDecompress, t.Logger, "-i", t.tmpJP2, "-r", "4", "-o", t.tmpPNGTest)
}
