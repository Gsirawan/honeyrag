package main

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/joho/godotenv"
)

// Styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFB347")).
			MarginBottom(1)

	honeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFD700"))

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00FF00"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF6B6B"))

	waitingStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#87CEEB"))

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666"))

	urlStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00BFFF")).
			Underline(true)

	progressStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFA500"))
)

// Step represents a setup/startup step
type Step struct {
	Name        string
	Status      string // "pending", "running", "done", "error"
	Description string
	Progress    string // For showing download progress etc.
}

// Model is the Bubble Tea model
type Model struct {
	steps       []Step
	spinner     spinner.Model
	currentStep int
	done        bool
	err         error
	baseDir     string
	quitting    bool
	ports       map[string]string
}

// Messages
type stepDoneMsg struct{ index int }
type stepErrorMsg struct {
	index int
	err   error
}
type stepProgressMsg struct {
	index   int
	message string
}
type startServicesMsg struct{}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func initialModel(baseDir string) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFD700"))

	// Load config
	envPath := filepath.Join(baseDir, "configs", ".env")
	godotenv.Load(envPath)

	ports := map[string]string{
		"ollama":   getEnv("OLLAMA_PORT", "11434"),
		"vllm":     getEnv("VLLM_PORT", "8000"),
		"lightrag": getEnv("LIGHTRAG_PORT", "9621"),
		"agno":     getEnv("AGNO_PORT", "8081"),
	}

	steps := []Step{
		{Name: "Ollama", Description: "Check/install Ollama", Status: "pending"},
		{Name: "Ollama Server", Description: "Start Ollama server", Status: "pending"},
		{Name: "Embedding Model", Description: "Check/pull nomic-embed-text", Status: "pending"},
		{Name: "vLLM Server", Description: "Start vLLM (loading model to GPU)", Status: "pending"},
		{Name: "LightRAG", Description: "Start RAG pipeline", Status: "pending"},
		{Name: "HoneyRAG Agent", Description: "Start web agent", Status: "pending"},
	}

	return Model{
		steps:       steps,
		spinner:     s,
		currentStep: 0,
		baseDir:     baseDir,
		ports:       ports,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.runStep(0))
}

func (m Model) runStep(index int) tea.Cmd {
	return func() tea.Msg {
		switch index {
		case 0:
			return m.checkInstallOllama(index)
		case 1:
			return m.startOllama(index)
		case 2:
			return m.pullEmbeddingModel(index)
		case 3:
			return m.startVLLM(index)
		case 4:
			return m.startLightRAG(index)
		case 5:
			return m.startAgent(index)
		}
		return stepDoneMsg{index: index}
	}
}

func (m Model) checkInstallOllama(index int) tea.Msg {
	// Check if ollama exists
	_, err := exec.LookPath("ollama")
	if err == nil {
		return stepDoneMsg{index: index}
	}

	// Install Ollama
	cmd := exec.Command("bash", "-c", "curl -fsSL https://ollama.ai/install.sh | sh")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return stepErrorMsg{index: index, err: fmt.Errorf("failed to install Ollama: %s", string(output))}
	}

	return stepDoneMsg{index: index}
}

func (m Model) pullEmbeddingModel(index int) tea.Msg {
	// Give Ollama a moment to fully initialize
	time.Sleep(2 * time.Second)

	// Check if model exists (retry a few times)
	for i := 0; i < 3; i++ {
		cmd := exec.Command("ollama", "list")
		output, err := cmd.Output()
		if err == nil && strings.Contains(string(output), "nomic-embed-text") {
			return stepDoneMsg{index: index}
		}
		time.Sleep(1 * time.Second)
	}

	// Pull the model
	cmd := exec.Command("ollama", "pull", "nomic-embed-text")
	err := cmd.Run()
	if err != nil {
		return stepErrorMsg{index: index, err: fmt.Errorf("failed to pull nomic-embed-text: %v", err)}
	}

	return stepDoneMsg{index: index}
}

func (m Model) startOllama(index int) tea.Msg {
	healthURL := fmt.Sprintf("http://localhost:%s/api/tags", m.ports["ollama"])

	// Check if already running
	if isHealthy(healthURL) {
		return stepDoneMsg{index: index}
	}

	// Start ollama serve
	cmd := exec.Command("ollama", "serve")
	cmd.Stdout = nil
	cmd.Stderr = nil
	err := cmd.Start()
	if err != nil {
		return stepErrorMsg{index: index, err: fmt.Errorf("failed to start Ollama: %v", err)}
	}

	// Wait for healthy
	if !waitForHealthy(healthURL, 30) {
		return stepErrorMsg{index: index, err: fmt.Errorf("Ollama failed to start")}
	}

	return stepDoneMsg{index: index}
}

func (m Model) startVLLM(index int) tea.Msg {
	healthURL := fmt.Sprintf("http://localhost:%s/v1/models", m.ports["vllm"])

	// Check if already running
	if isHealthy(healthURL) {
		return stepDoneMsg{index: index}
	}

	// Get model from env
	model := getEnv("VLLM_MODEL", "Qwen/Qwen3-8B")
	gpuUtil := getEnv("VLLM_GPU_MEMORY_UTILIZATION", "0.8")
	maxLen := getEnv("VLLM_MAX_MODEL_LEN", "8192")

	// Start vLLM
	cmdStr := fmt.Sprintf("cd %s && source .venv/bin/activate && vllm serve %s --port %s --gpu-memory-utilization %s --max-model-len %s --enforce-eager",
		m.baseDir, model, m.ports["vllm"], gpuUtil, maxLen)

	cmd := exec.Command("bash", "-c", cmdStr)
	cmd.Dir = m.baseDir
	cmd.Stdout = nil
	cmd.Stderr = nil
	err := cmd.Start()
	if err != nil {
		return stepErrorMsg{index: index, err: fmt.Errorf("failed to start vLLM: %v", err)}
	}

	// Wait for healthy - longer timeout for model download
	if !waitForHealthy(healthURL, 300) { // 5 min timeout for model download
		return stepErrorMsg{index: index, err: fmt.Errorf("vLLM failed to start (timeout)")}
	}

	return stepDoneMsg{index: index}
}

func (m Model) startLightRAG(index int) tea.Msg {
	healthURL := fmt.Sprintf("http://localhost:%s/health", m.ports["lightrag"])

	// Check if already running
	if isHealthy(healthURL) {
		return stepDoneMsg{index: index}
	}

	// Start LightRAG
	cmdStr := fmt.Sprintf("cd %s/services/lightrag && source %s/.venv/bin/activate && lightrag-server",
		m.baseDir, m.baseDir)

	cmd := exec.Command("bash", "-c", cmdStr)
	cmd.Dir = filepath.Join(m.baseDir, "services", "lightrag")
	cmd.Stdout = nil
	cmd.Stderr = nil
	err := cmd.Start()
	if err != nil {
		return stepErrorMsg{index: index, err: fmt.Errorf("failed to start LightRAG: %v", err)}
	}

	// Wait for healthy
	if !waitForHealthy(healthURL, 60) {
		return stepErrorMsg{index: index, err: fmt.Errorf("LightRAG failed to start")}
	}

	return stepDoneMsg{index: index}
}

func (m Model) startAgent(index int) tea.Msg {
	healthURL := fmt.Sprintf("http://localhost:%s/health", m.ports["agno"])

	// Check if already running
	if isHealthy(healthURL) {
		return stepDoneMsg{index: index}
	}

	// Start agent
	cmdStr := fmt.Sprintf("cd %s/services/agno && source %s/.venv/bin/activate && uvicorn app:app --host 0.0.0.0 --port %s",
		m.baseDir, m.baseDir, m.ports["agno"])

	cmd := exec.Command("bash", "-c", cmdStr)
	cmd.Dir = filepath.Join(m.baseDir, "services", "agno")
	cmd.Stdout = nil
	cmd.Stderr = nil
	err := cmd.Start()
	if err != nil {
		return stepErrorMsg{index: index, err: fmt.Errorf("failed to start Agent: %v", err)}
	}

	// Wait for healthy
	if !waitForHealthy(healthURL, 30) {
		return stepErrorMsg{index: index, err: fmt.Errorf("Agent failed to start")}
	}

	return stepDoneMsg{index: index}
}

func isHealthy(url string) bool {
	client := http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200
}

func waitForHealthy(url string, timeoutSeconds int) bool {
	for i := 0; i < timeoutSeconds; i++ {
		if isHealthy(url) {
			return true
		}
		time.Sleep(1 * time.Second)
	}
	return false
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case stepDoneMsg:
		m.steps[msg.index].Status = "done"
		m.currentStep++
		if m.currentStep >= len(m.steps) {
			m.done = true
			return m, nil
		}
		m.steps[m.currentStep].Status = "running"
		return m, m.runStep(m.currentStep)

	case stepErrorMsg:
		m.steps[msg.index].Status = "error"
		m.err = msg.err
		return m, nil

	case stepProgressMsg:
		m.steps[msg.index].Progress = msg.message
		return m, nil
	}

	return m, nil
}

func (m Model) View() string {
	var b strings.Builder

	// Header
	honey := honeyStyle.Render("🍯")
	title := titleStyle.Render(fmt.Sprintf("\n%s HoneyRAG - Local RAG Stack %s", honey, honey))
	b.WriteString(title)
	b.WriteString("\n\n")

	// Steps status
	for i, step := range m.steps {
		var icon string
		var status string

		switch step.Status {
		case "pending":
			icon = dimStyle.Render("○")
			status = dimStyle.Render(step.Description)
		case "running":
			icon = m.spinner.View()
			status = waitingStyle.Render(step.Description + "...")
		case "done":
			icon = successStyle.Render("●")
			status = successStyle.Render(step.Description)
		case "error":
			icon = errorStyle.Render("✗")
			status = errorStyle.Render(step.Description)
		}

		line := fmt.Sprintf("  %s %s: %s", icon, step.Name, status)
		b.WriteString(line)
		b.WriteString("\n")

		// Show progress if any
		if step.Progress != "" && step.Status == "running" {
			b.WriteString(dimStyle.Render("    └─ "))
			b.WriteString(progressStyle.Render(step.Progress))
			b.WriteString("\n")
		}

		// Show current activity
		if i == m.currentStep && step.Status == "running" && step.Progress == "" {
			hint := ""
			switch i {
			case 0:
				hint = "checking installation..."
			case 1:
				hint = "waiting for server..."
			case 2:
				hint = "checking model (pulls if needed)..."
			case 3:
				hint = "loading model into GPU (~30-60s)..."
			case 4:
				hint = "initializing RAG pipeline..."
			case 5:
				hint = "starting web interface..."
			}
			if hint != "" {
				b.WriteString(dimStyle.Render("    └─ "))
				b.WriteString(waitingStyle.Render(hint))
				b.WriteString("\n")
			}
		}
	}

	b.WriteString("\n")

	// Status message
	if m.err != nil {
		b.WriteString(errorStyle.Render(fmt.Sprintf("Error: %v", m.err)))
		b.WriteString("\n\n")
		b.WriteString(dimStyle.Render("Check logs and try again. Press 'q' to quit."))
	} else if m.done {
		b.WriteString(successStyle.Render("✨ All services running!"))
		b.WriteString("\n\n")
		b.WriteString(honeyStyle.Render("  🍯 Sweet endpoints ready:"))
		b.WriteString("\n\n")
		b.WriteString(fmt.Sprintf("     Agent UI:     %s\n", urlStyle.Render(fmt.Sprintf("http://localhost:%s", m.ports["agno"]))))
		b.WriteString(fmt.Sprintf("     LightRAG UI:  %s\n", urlStyle.Render(fmt.Sprintf("http://localhost:%s", m.ports["lightrag"]))))
		b.WriteString(fmt.Sprintf("     vLLM API:     %s\n", urlStyle.Render(fmt.Sprintf("http://localhost:%s", m.ports["vllm"]))))
		b.WriteString("\n")
		b.WriteString(dimStyle.Render("  Press 'q' to stop all services and quit"))
	} else {
		b.WriteString(dimStyle.Render("  Setting up... Press 'q' to cancel"))
	}

	b.WriteString("\n")

	return b.String()
}

func main() {
	// Get base directory (where the binary is, or current dir)
	baseDir, err := os.Getwd()
	if err != nil {
		fmt.Println("Error getting current directory:", err)
		os.Exit(1)
	}

	// Verify we're in the right directory
	if _, err := os.Stat(filepath.Join(baseDir, "configs", ".env")); os.IsNotExist(err) {
		if _, err := os.Stat(filepath.Join(baseDir, "configs", ".env.example")); os.IsNotExist(err) {
			fmt.Println("Error: Run this from the honeyrag directory")
			fmt.Println("Expected to find: configs/.env or configs/.env.example")
			os.Exit(1)
		}
	}

	// Start the TUI
	p := tea.NewProgram(initialModel(baseDir))
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}
