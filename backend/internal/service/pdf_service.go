package service

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// PDFService 使用 wkhtmltopdf 将 HTML 导出为 PDF。
type PDFService struct {
	binPath string
	workDir string
	logger  *slog.Logger
}

// NewPDFService 构造 PDF 导出服务。
func NewPDFService(logger *slog.Logger) *PDFService {
	return &PDFService{
		binPath: "wkhtmltopdf",
		workDir: os.TempDir(),
		logger:  logger,
	}
}

// Generate 将 HTML 写入临时文件并调用 wkhtmltopdf 生成 PDF，返回生成文件路径。
func (s *PDFService) Generate(html string) (string, error) {
	if _, err := exec.LookPath(s.binPath); err != nil {
		return "", fmt.Errorf("wkhtmltopdf not found: %w", err)
	}
	if err := os.MkdirAll(s.workDir, 0o755); err != nil {
		return "", fmt.Errorf("create pdf workdir: %w", err)
	}
	base := fmt.Sprintf("contract-%d", time.Now().UnixNano())
	htmlFile := filepath.Join(s.workDir, base+".html")
	pdfFile := filepath.Join(s.workDir, base+".pdf")
	if err := os.WriteFile(htmlFile, []byte(html), 0o644); err != nil {
		return "", fmt.Errorf("write html temp file: %w", err)
	}
	defer func() {
		if err := os.Remove(htmlFile); err != nil && !os.IsNotExist(err) {
			s.logger.Warn("remove html temp file failed", "file", htmlFile, "error", err)
		}
	}()

	cmd := exec.Command(s.binPath, "--encoding", "utf-8", "--quiet", htmlFile, pdfFile)
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("run wkhtmltopdf: %w: %s", err, string(output))
	}
	s.logger.Info("pdf generated", "file", pdfFile)
	return pdfFile, nil
}

// Cleanup 删除生成的临时 PDF 文件。
func (s *PDFService) Cleanup(path string) {
	if path == "" {
		return
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		s.logger.Warn("remove pdf file failed", "file", path, "error", err)
	}
}
