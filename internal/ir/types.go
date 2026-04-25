package ir

type Engine string

const (
	EngineHAProxy Engine = "haproxy"
	EngineNginx   Engine = "nginx"
)

type Model struct {
	Version      int            `json:"version"`
	ID           string         `json:"id"`
	Name         string         `json:"name"`
	Description  string         `json:"description,omitempty"`
	Engines      []Engine       `json:"engines"`
	Frontends    []Frontend     `json:"frontends"`
	Backends     []Backend      `json:"backends"`
	Servers      []Server       `json:"servers"`
	Rules        []Rule         `json:"rules"`
	TLSProfiles  []TLSProfile   `json:"tls_profiles"`
	HealthChecks []HealthCheck  `json:"health_checks"`
	RateLimits   []RateLimit    `json:"rate_limits"`
	Caches       []CachePolicy  `json:"caches"`
	Loggers      []Logger       `json:"loggers"`
	OpaqueBlocks []OpaqueBlock  `json:"opaque_blocks,omitempty"`
	View         ModelView      `json:"view"`
	Metadata     map[string]any `json:"metadata,omitempty"`
}

type Point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type EntityView struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type ModelView struct {
	Zoom float64 `json:"zoom"`
	Pan  Point   `json:"pan"`
}

type Frontend struct {
	ID             string     `json:"id"`
	Name           string     `json:"name"`
	Bind           string     `json:"bind"`
	Protocol       string     `json:"protocol"`
	TLSID          string     `json:"tls_id,omitempty"`
	Rules          []string   `json:"rules,omitempty"`
	DefaultBackend string     `json:"default_backend,omitempty"`
	View           EntityView `json:"view"`
}

type Backend struct {
	ID            string     `json:"id"`
	Name          string     `json:"name"`
	Algorithm     string     `json:"algorithm"`
	HealthCheckID string     `json:"health_check_id,omitempty"`
	Servers       []string   `json:"servers"`
	View          EntityView `json:"view"`
}

type Server struct {
	ID      string         `json:"id"`
	Name    string         `json:"name,omitempty"`
	Address string         `json:"address"`
	Port    int            `json:"port"`
	Weight  int            `json:"weight"`
	MaxConn int            `json:"max_conn,omitempty"`
	Attrs   map[string]any `json:"attrs,omitempty"`
}

type Rule struct {
	ID        string     `json:"id"`
	Name      string     `json:"name,omitempty"`
	Predicate Predicate  `json:"predicate"`
	Action    RuleAction `json:"action"`
	View      EntityView `json:"view"`
}

type Predicate struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type RuleAction struct {
	Type      string `json:"type"`
	BackendID string `json:"backend_id,omitempty"`
	Location  string `json:"location,omitempty"`
	Status    int    `json:"status,omitempty"`
}

type TLSProfile struct {
	ID         string   `json:"id"`
	Name       string   `json:"name,omitempty"`
	CertPath   string   `json:"cert_path"`
	KeyPath    string   `json:"key_path,omitempty"`
	Ciphers    string   `json:"ciphers,omitempty"`
	MinVersion string   `json:"min_version,omitempty"`
	ALPN       []string `json:"alpn,omitempty"`
}

type HealthCheck struct {
	ID             string `json:"id"`
	Name           string `json:"name,omitempty"`
	Type           string `json:"type"`
	Path           string `json:"path,omitempty"`
	ExpectedStatus []int  `json:"expected_status,omitempty"`
	IntervalMS     int    `json:"interval_ms"`
	TimeoutMS      int    `json:"timeout_ms"`
	Rise           int    `json:"rise"`
	Fall           int    `json:"fall"`
}

type RateLimit struct {
	ID       string `json:"id"`
	Name     string `json:"name,omitempty"`
	Key      string `json:"key"`
	PeriodMS int    `json:"period_ms"`
	Requests int    `json:"requests"`
}

type CachePolicy struct {
	ID      string `json:"id"`
	Name    string `json:"name,omitempty"`
	Zone    string `json:"zone"`
	Path    string `json:"path"`
	MaxSize string `json:"max_size,omitempty"`
}

type Logger struct {
	ID          string `json:"id"`
	Name        string `json:"name,omitempty"`
	Destination string `json:"destination"`
	Format      string `json:"format,omitempty"`
}

type OpaqueBlock struct {
	ID      string   `json:"id"`
	Section string   `json:"section"`
	Lines   []string `json:"lines"`
	Anchor  string   `json:"anchor,omitempty"`
}

func EmptyModel(id, name, description string, engines []Engine) *Model {
	if len(engines) == 0 {
		engines = []Engine{EngineHAProxy}
	}
	return &Model{
		Version:      1,
		ID:           id,
		Name:         name,
		Description:  description,
		Engines:      engines,
		Frontends:    []Frontend{},
		Backends:     []Backend{},
		Servers:      []Server{},
		Rules:        []Rule{},
		TLSProfiles:  []TLSProfile{},
		HealthChecks: []HealthCheck{},
		RateLimits:   []RateLimit{},
		Caches:       []CachePolicy{},
		Loggers:      []Logger{},
		OpaqueBlocks: []OpaqueBlock{},
		View:         ModelView{Zoom: 1, Pan: Point{}},
	}
}
