package limoni

// testApp builds an App from an appConfig, the way the options would.
func testApp(term *Terminal, cfg appConfig) *App {
	return &App{term: term, cfg: cfg, wakeup: make(chan struct{}, 1)}
}
