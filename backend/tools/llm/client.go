package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type EvidenceItem struct {
	Page    int32  `json:"page"`
	Quote   string `json:"quote"`
	Section string `json:"section"`
}

type ExtractedField struct {
	Value      string         `json:"value"`
	SourceType string         `json:"source_type"` // explicit, inferred, not_reported
	Evidence   []EvidenceItem `json:"evidence"`
}

type PaperExtractionResult struct {
	Title             string           `json:"title"`
	Authors           []string         `json:"authors"`
	PublicationYear   int32            `json:"publication_year"`
	Journal           string           `json:"journal"`
	DOI               string           `json:"doi"`
	Abstract          string           `json:"abstract"`
	Keywords          []string         `json:"keywords"`
	Language          string           `json:"language"`
	PaperType         string           `json:"paper_type"`
	ResearchProblem   ExtractedField   `json:"research_problem"`
	ResearchObjective ExtractedField   `json:"research_objective"`
	Contributions     ExtractedField   `json:"contributions"`
	Methodology       ExtractedField   `json:"methodology"`
	Datasets          ExtractedField   `json:"datasets"`
	ExpSettings       ExtractedField   `json:"exp_settings"`
	Metrics           ExtractedField   `json:"metrics"`
	Results           ExtractedField   `json:"results"`
	Limitations       ExtractedField   `json:"limitations"`
	FutureWork        ExtractedField   `json:"future_work"`
	CustomFields      map[string]ExtractedField `json:"custom_fields,omitempty"`
}

type ComparisonDiffResult struct {
	ObjectiveComparison string `json:"objective_comparison"`
	MethodDifferences   string `json:"method_differences"`
	DatasetDifferences  string `json:"dataset_differences"`
	SettingDifferences  string `json:"setting_differences"`
	ResultDifferences   string `json:"result_differences"`
	CommonLimitations   string `json:"common_limitations"`
	EvolutionTrend      string `json:"evolution_trend"`
	SummaryMarkdown     string `json:"summary_markdown"`
}

type Client struct {
	apiKey     string
	baseURL    string
	models     []string
	httpClient *http.Client
}

func NewClient(apiKey, baseURL, preferredModel string) *Client {
	if apiKey == "" {
		apiKey = os.Getenv("DASHSCOPE_API_KEY")
	}
	if baseURL == "" {
		baseURL = "https://dashscope.aliyuncs.com/compatible-mode/v1/chat/completions"
	}
	models := []string{"qwen-flash", "qwen-plus", "qwen3.5-flash", "qwen-turbo"}
	if preferredModel != "" {
		models = append([]string{preferredModel}, models...)
	}
	return &Client{
		apiKey:  apiKey,
		baseURL: baseURL,
		models:  models,
		httpClient: &http.Client{
			Timeout: 40 * time.Second,
		},
	}
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model          string            `json:"model"`
	Messages       []chatMessage     `json:"messages"`
	ResponseFormat map[string]string `json:"response_format,omitempty"`
	Temperature    float64           `json:"temperature"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Code    string `json:"code"`
	} `json:"error,omitempty"`
}

func (c *Client) CallJSON(ctx context.Context, systemPrompt, userPrompt string, target any) error {
	if c.apiKey == "" {
		return fmt.Errorf("DASHSCOPE_API_KEY is empty")
	}

	var lastErr error
	for _, model := range c.models {
		attemptCtx, attemptCancel := context.WithTimeout(ctx, 35*time.Second)
		reqPayload := chatRequest{
			Model: model,
			Messages: []chatMessage{
				{Role: "system", Content: systemPrompt},
				{Role: "user", Content: userPrompt},
			},
			ResponseFormat: map[string]string{"type": "json_object"},
			Temperature:    0.2,
		}
		bodyBytes, _ := json.Marshal(reqPayload)

		req, err := http.NewRequestWithContext(attemptCtx, http.MethodPost, c.baseURL, bytes.NewReader(bodyBytes))
		if err != nil {
			attemptCancel()
			lastErr = err
			continue
		}
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			attemptCancel()
			lastErr = err
			continue
		}
		respBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		attemptCancel()

		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("model %s HTTP %d: %s", model, resp.StatusCode, string(respBytes))
			continue
		}

		var cr chatResponse
		if err := json.Unmarshal(respBytes, &cr); err != nil {
			lastErr = err
			continue
		}
		if len(cr.Choices) == 0 {
			lastErr = fmt.Errorf("model %s returned empty choices", model)
			continue
		}

		content := cleanJSONOutput(cr.Choices[0].Message.Content)
		if err := json.Unmarshal([]byte(content), target); err != nil {
			lastErr = fmt.Errorf("invalid json from model %s: %w", model, err)
			continue
		}
		return nil
	}
	return lastErr
}

func cleanJSONOutput(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	return strings.TrimSpace(s)
}

func (c *Client) ExtractPaperInfo(ctx context.Context, pageTexts map[int]string, fallbackTitle, fallbackAbstract string) (*PaperExtractionResult, error) {
	var promptContent strings.Builder
	totalLen := 0
	for p := 1; p <= len(pageTexts); p++ {
		txt := pageTexts[p]
		if txt == "" {
			continue
		}
		if len(txt) > 3500 {
			txt = txt[:3500]
		}
		chunk := fmt.Sprintf("=== [Page %d] ===\n%s\n\n", p, txt)
		if totalLen+len(chunk) > 22000 {
			break
		}
		promptContent.WriteString(chunk)
		totalLen += len(chunk)
	}

	if promptContent.Len() == 0 {
		promptContent.WriteString(fmt.Sprintf("=== [Page 1] ===\nTitle: %s\nAbstract: %s\n", fallbackTitle, fallbackAbstract))
	}

	sysPrompt := `You are a rigorous academic paper analysis engine for PaperCompare.
Analyze the provided paper pages and output a valid JSON object matching this exact schema:
{
  "title": "Paper title",
  "authors": ["Author 1", "Author 2"],
  "publication_year": 2024,
  "journal": "Conference or Journal name",
  "doi": "10.xxxx/...",
  "abstract": "Abstract summary",
  "keywords": ["Keyword1", "Keyword2"],
  "language": "English or Chinese",
  "paper_type": "Research Paper / Survey / System Paper",
  "research_problem": {"value": "Clear description in Chinese", "source_type": "explicit", "evidence": [{"page": 1, "quote": "Exact quote from paper", "section": "Introduction"}]},
  "research_objective": {"value": "Clear description in Chinese", "source_type": "explicit", "evidence": [{"page": 1, "quote": "Exact quote", "section": "Introduction"}]},
  "contributions": {"value": "Bullet points in Chinese of author-stated contributions", "source_type": "explicit", "evidence": [{"page": 2, "quote": "Exact quote", "section": "Introduction"}]},
  "methodology": {"value": "Detailed algorithm, architecture, loss function, and workflow in Chinese", "source_type": "explicit", "evidence": [{"page": 3, "quote": "Exact quote", "section": "Methodology"}]},
  "datasets": {"value": "Dataset names, sizes, splits, preprocessing in Chinese", "source_type": "explicit", "evidence": [{"page": 5, "quote": "Exact quote", "section": "Experiments"}]},
  "exp_settings": {"value": "Backbone, hyperparameters, clients/GPUs, learning rate, optimizer in Chinese (mark unreported items clearly)", "source_type": "explicit", "evidence": [{"page": 5, "quote": "Exact quote", "section": "Experiments"}]},
  "metrics": {"value": "Evaluation metrics used (e.g., Accuracy, F1, Communication Cost)", "source_type": "explicit", "evidence": [{"page": 6, "quote": "Exact quote", "section": "Experiments"}]},
  "results": {"value": "Key quantitative results and ablation findings with dataset context in Chinese", "source_type": "explicit", "evidence": [{"page": 6, "quote": "Exact quote", "section": "Results"}]},
  "limitations": {"value": "Author-reported limitations in Chinese, or '论文未明确报告局限性'", "source_type": "explicit", "evidence": [{"page": 8, "quote": "Exact quote", "section": "Conclusion"}]},
  "future_work": {"value": "Future research directions in Chinese", "source_type": "explicit", "evidence": [{"page": 8, "quote": "Exact quote", "section": "Conclusion"}]}
}
Rules:
1. source_type must be one of: "explicit", "inferred", "not_reported".
2. Never fabricate metrics or datasets. If a field is not reported in the text, set value to "未报告" and source_type to "not_reported".
3. Keep analysis values concise, accurate, and in Chinese (while preserving original English technical terms/acronyms and exact quotes).`

	var res PaperExtractionResult
	err := c.CallJSON(ctx, sysPrompt, promptContent.String(), &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (c *Client) ExtractComparisonFields(ctx context.Context, paperTitle, paperContext string, fields []string) (map[string]ExtractedField, error) {
	sysPrompt := `You are a scientific literature comparison assistant.
Given the paper content and a list of comparison dimension keys, extract the value for each dimension.
Return a JSON object where each key is the dimension key, and the value is:
{
  "value": "Extracted concise answer in Chinese (or '未报告' if absent)",
  "source_type": "explicit | inferred | not_reported",
  "evidence": [{"page": 1, "quote": "Short supporting quote", "section": "Section"}]
}`

	userPrompt := fmt.Sprintf("Paper Title: %s\nDimensions to extract: %s\n\nPaper Content:\n%s",
		paperTitle, strings.Join(fields, ", "), paperContext)

	result := make(map[string]ExtractedField)
	err := c.CallJSON(ctx, sysPrompt, userPrompt, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) AnalyzeComparisonMatrix(ctx context.Context, topic string, matrixJSON string) (*ComparisonDiffResult, error) {
	sysPrompt := `You are a senior scientific research advisor for PaperCompare.
Analyze the multi-paper comparison matrix provided by the user and output a JSON object with this exact schema:
{
  "objective_comparison": "Compare the core research problems and objectives across the papers (in Chinese)",
  "method_differences": "Compare the algorithms, model architectures, knowledge transfer or technical routes (in Chinese)",
  "dataset_differences": "Compare the datasets, data distributions (e.g., Non-IID settings), and scales (in Chinese)",
  "setting_differences": "Compare experimental setups, backbones, client counts, and hardware/hyperparameters (in Chinese)",
  "result_differences": "Compare reported performance metrics; note where experimental settings differ so numbers cannot be naively ranked (in Chinese)",
  "common_limitations": "Synthesize author-reported limitations and remaining open research gaps (in Chinese)",
  "evolution_trend": "Describe the chronological and methodological evolution across these papers (in Chinese)",
  "summary_markdown": "A well-structured Markdown executive summary of the multi-paper comparison suitable for a literature review report (in Chinese)"
}`

	userPrompt := fmt.Sprintf("Research Topic: %s\nComparison Matrix Data:\n%s", topic, matrixJSON)
	var res ComparisonDiffResult
	if err := c.CallJSON(ctx, sysPrompt, userPrompt, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

