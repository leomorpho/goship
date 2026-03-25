package agenteval

import (
	"encoding/json"
	"os"
	"sort"
)

type TaskPack struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Tasks       []TaskSpec `json:"tasks"`
}

type TaskSpec struct {
	ID                 string   `json:"id"`
	Title              string   `json:"title"`
	RequiredSurfaces   []string `json:"required_surfaces"`
	RequiredTools      []string `json:"required_tools"`
	DisallowedSurfaces []string `json:"disallowed_surfaces"`
}

type TaskAttempt struct {
	FirstSurface   string
	UsedSurfaces   []string
	UsedTools      []string
	Completed      bool
	AvoidedDeadEnds bool
}

type TaskScore struct {
	TaskID                string
	Score                 int
	Passed                bool
	MissingRequiredTools  []string
	MissingRequiredSurfaces []string
	HitDisallowedSurfaces []string
}

type PackSummary struct {
	PackName     string
	TotalTasks   int
	PassedTasks  int
	SuccessRate  float64
	AverageScore float64
	Results      []TaskScore
}

func LoadColdStartTaskPack(path string) (TaskPack, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return TaskPack{}, err
	}
	var pack TaskPack
	if err := json.Unmarshal(b, &pack); err != nil {
		return TaskPack{}, err
	}
	return pack, nil
}

func EvaluateTaskAttempt(task TaskSpec, attempt TaskAttempt) TaskScore {
	score := 0
	result := TaskScore{TaskID: task.ID}

	if inList(task.RequiredSurfaces, attempt.FirstSurface) {
		score += 25
	}
	if attempt.Completed {
		score += 30
	}
	if attempt.AvoidedDeadEnds {
		score += 15
	}

	result.MissingRequiredSurfaces = missing(task.RequiredSurfaces, attempt.UsedSurfaces)
	if len(result.MissingRequiredSurfaces) == 0 {
		score += 15
	}

	result.MissingRequiredTools = missing(task.RequiredTools, attempt.UsedTools)
	if len(result.MissingRequiredTools) == 0 {
		score += 15
	}

	result.HitDisallowedSurfaces = intersect(task.DisallowedSurfaces, attempt.UsedSurfaces)
	if len(result.HitDisallowedSurfaces) > 0 {
		score -= 20
	}

	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	result.Score = score
	result.Passed = score >= 70 && attempt.Completed
	return result
}

func EvaluatePack(pack TaskPack, attempts map[string]TaskAttempt) PackSummary {
	summary := PackSummary{PackName: pack.Name, TotalTasks: len(pack.Tasks)}
	if len(pack.Tasks) == 0 {
		return summary
	}

	results := make([]TaskScore, 0, len(pack.Tasks))
	totalScore := 0
	passed := 0
	for _, task := range pack.Tasks {
		result := EvaluateTaskAttempt(task, attempts[task.ID])
		results = append(results, result)
		totalScore += result.Score
		if result.Passed {
			passed++
		}
	}

	summary.Results = results
	summary.PassedTasks = passed
	summary.SuccessRate = float64(passed) / float64(len(pack.Tasks))
	summary.AverageScore = float64(totalScore) / float64(len(pack.Tasks))
	return summary
}

func inList(items []string, needle string) bool {
	for _, item := range items {
		if item == needle {
			return true
		}
	}
	return false
}

func missing(required []string, used []string) []string {
	missing := make([]string, 0)
	for _, req := range required {
		if !inList(used, req) {
			missing = append(missing, req)
		}
	}
	sort.Strings(missing)
	return missing
}

func intersect(left []string, right []string) []string {
	hits := make([]string, 0)
	for _, item := range left {
		if inList(right, item) {
			hits = append(hits, item)
		}
	}
	sort.Strings(hits)
	return hits
}
