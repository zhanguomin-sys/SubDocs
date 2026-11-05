package connect

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// ModelsState is the outcome of GET /v1/models for a key.
type ModelsState int

const (
	ModelsImage ModelsState = iota // lists at least one gpt-image model
	ModelsNoImage
	ModelsRejected
	ModelsTimeout
	ModelsOffline
	ModelsServerError
	ModelsNoBalance // the key is fine but the account balance is $0
)

const (
	modelsPath      = "/v1/models"
	maxModelsAnswer = 8 << 20
)

// ListModels returns the model IDs key may call, from GET /v1/models. The
// answer is the OpenAI shape (data[].id); the Codex shape (models[].slug) is
// read too. On success the state is ModelsImage; the caller decides what the
// list means.
func ListModels(ctx context.Context, client *http.Client, baseURL, key string) ([]string, ModelsState, int) {
	ctx, cancel := context.WithTimeout(ctx, usageTimeout)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+modelsPath, nil)
	if err != nil {
		return nil, ModelsOffline, 0
	}
	request.Header.Set("Authorization", "Bearer "+key)
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", "MyAutoAPI-Quick-Connect/"+Version)
	response, err := client.Do(request)
	if err != nil {
		if isTimeout(err) {
			return nil, ModelsTimeout, 0
		}
		return nil, ModelsOffline, 0
	}
	defer response.Body.Close()
	switch response.StatusCode {
	case http.StatusOK:
	case http.StatusUnauthorized, http.StatusForbidden:
		// sub2api refuses every endpoint but /v1/usage while the balance
		// is $0, so a good key looks rejected here; its error code tells.
		body, _ := io.ReadAll(io.LimitReader(response.Body, 64<<10))
		if strings.Contains(string(body), "INSUFFICIENT_BALANCE") {
			return nil, ModelsNoBalance, response.StatusCode
		}
		return nil, ModelsRejected, response.StatusCode
	default:
		return nil, ModelsServerError, response.StatusCode
	}
	var answer struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
		Models []struct {
			Slug string `json:"slug"`
		} `json:"models"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, maxModelsAnswer)).Decode(&answer); err != nil {
		if isTimeout(err) {
			return nil, ModelsTimeout, 0
		}
		return nil, ModelsServerError, response.StatusCode
	}
	var ids []string
	for _, model := range answer.Data {
		ids = append(ids, model.ID)
	}
	for _, model := range answer.Models {
		ids = append(ids, model.Slug)
	}
	return ids, ModelsImage, response.StatusCode
}

// CheckImageModels asks which models key may use. A key whose group lists no
// gpt-image model certainly cannot draw; any listed one is good enough.
func CheckImageModels(ctx context.Context, client *http.Client, baseURL, key string) (ModelsState, int) {
	ids, state, code := ListModels(ctx, client, baseURL, key)
	if state != ModelsImage {
		return state, code
	}
	for _, id := range ids {
		if isImageModel(id) {
			return ModelsImage, code
		}
	}
	return ModelsNoImage, code
}

func isImageModel(id string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(id)), ImageModelPrefix)
}

// claudeModelPattern is the official Anthropic model ID form: lower case,
// the family, a major version, an optional minor version and an optional
// eight-digit snapshot date, all joined by hyphens. Relays sometimes list
// look-alikes such as claude-opus-5.5, Claude-Opus-5-5, claude-opus-5-5[1m]
// or claude-opus-5-5-thinking; Claude Desktop rejects the whole model list
// when one entry is not a real ID, so those are dropped, never rewritten (the
// corrected name might not exist on the relay).
var claudeModelPattern = regexp.MustCompile(`^claude-(fable|opus|sonnet|haiku)-(\d{1,2})(?:-(\d{1,2}))?(?:-(\d{8}))?$`)

// claudeFamilies is the menu order in Claude Desktop; the first model in the
// list is its default.
var claudeFamilies = map[string]int{"fable": 0, "opus": 1, "sonnet": 2, "haiku": 3}

// maxClaudeModels is the most Claude Desktop accepts in inferenceModels.
const maxClaudeModels = 200

// ClaudeDesktopModels keeps every listed model of the four Claude families
// in the official ID form. A dated snapshot is dropped when the same version
// is also listed without a date. Order: Fable, Opus, Sonnet, Haiku; newer
// versions first within a family.
func ClaudeDesktopModels(ids []string) []string {
	type model struct {
		id                  string
		family, major, minr int
		date                string
	}
	var models []model
	seen := map[string]bool{}
	undated := map[string]bool{}
	for _, raw := range ids {
		id := strings.TrimSpace(raw)
		match := claudeModelPattern.FindStringSubmatch(id)
		if match == nil || seen[id] {
			continue
		}
		seen[id] = true
		major, _ := strconv.Atoi(match[2])
		minor := -1
		if match[3] != "" {
			minor, _ = strconv.Atoi(match[3])
		}
		models = append(models, model{id: id, family: claudeFamilies[match[1]], major: major, minr: minor, date: match[4]})
		if match[4] == "" {
			undated[id] = true
		}
	}
	kept := models[:0]
	for _, m := range models {
		if m.date != "" && undated[strings.TrimSuffix(m.id, "-"+m.date)] {
			continue
		}
		kept = append(kept, m)
	}
	sort.SliceStable(kept, func(i, j int) bool {
		a, b := kept[i], kept[j]
		if a.family != b.family {
			return a.family < b.family
		}
		if a.major != b.major {
			return a.major > b.major
		}
		if a.minr != b.minr {
			return a.minr > b.minr
		}
		return a.date > b.date
	})
	result := make([]string, 0, len(kept))
	for _, m := range kept {
		if len(result) == maxClaudeModels {
			break
		}
		result = append(result, m.id)
	}
	return result
}
