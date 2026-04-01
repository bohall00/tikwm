package tikwm

import (
    "fmt"
    "sync"
)

type API struct {
    fsClient *FlareSolverrClient
    workers  int
}

func NewAPI(fsURL string, workers int) *API {
    return &API{
        fsClient: NewFlareSolverrClient(fsURL),
        workers:  workers,
    }
}

func (api *API) Initialize() error {
    // Warm FlareSolverr once at startup
    return api.fsClient.EnsureSession()
}

func (api *API) ProcessTargets(targets []string) {
    var wg sync.WaitGroup
    jobs := make(chan string)

    // Workers
    for i := 0; i < api.workers; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for target := range jobs {
                api.handleTarget(target)
            }
        }()
    }

    for _, t := range targets {
        jobs <- t
    }
    close(jobs)

    wg.Wait()
}

func (api *API) handleTarget(target string) {
    url := fmt.Sprintf("https://tikwm.com/@%s", target)

    html, err := api.fsClient.Get(url)
    if err != nil {
        fmt.Printf("✗ Failed to process target '%s': %v\n", target, err)
        return
    }

    // continue scraping/parsing from html here
    fmt.Printf("✓ Target processed: %s (%d bytes)\n", target, len(html))
}

func (api *API) Shutdown() {
    api.fsClient.DestroySession()
}
