package ai

// ChatRequest — запрос к DeepSeek API (OpenAI-совместимый формат).
type ChatRequest struct {
	Model          string          `json:"model"`
	Messages       []ChatMessage   `json:"messages"`
	Temperature    float64         `json:"temperature"`
	ResponseFormat *ResponseFormat `json:"response_format,omitempty"`
}

// ChatMessage — сообщение в формате OpenAI.
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ResponseFormat — формат ответа (json_object для гарантии JSON).
type ResponseFormat struct {
	Type string `json:"type"`
}

// ChatResponse — ответ DeepSeek API.
type ChatResponse struct {
	Choices []ChatChoice `json:"choices"`
	Error   *APIError    `json:"error"`
}

// ChatChoice — один вариант ответа.
type ChatChoice struct {
	Message ChatMessage `json:"message"`
}

// APIError — ошибка API.
type APIError struct {
	Message string `json:"message"`
}

// aiWorkoutResponse — структура JSON-ответа AI для тренировки.
type aiWorkoutResponse struct {
	Title       string       `json:"title"`
	MuscleGroup string       `json:"muscle_group"`
	Duration    string       `json:"duration"`
	Description string       `json:"description"`
	Exercises   []aiExercise `json:"exercises"`
}

// aiExercise — одно упражнение в ответе AI.
type aiExercise struct {
	Name string `json:"name"`
	Sets string `json:"sets"`
	Reps string `json:"reps"`
}

// aiNutritionResponse — структура JSON-ответа AI для питания.
type aiNutritionResponse struct {
	Breakfast string   `json:"breakfast"`
	Lunch     string   `json:"lunch"`
	Dinner    string   `json:"dinner"`
	Snacks    []string `json:"snacks"`
	Calories  string   `json:"calories"`
	Protein   string   `json:"protein"`
	Fat       string   `json:"fat"`
	Carbs     string   `json:"carbs"`
	WaterMl   string   `json:"water_ml"`
}

// aiMotivationResponse — структура JSON-ответа AI для мотивации.
type aiMotivationResponse struct {
	Text string `json:"text"`
}
