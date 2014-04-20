package main

import (
    "bytes"
    "code.google.com/p/go-uuid/uuid"
    "encoding/json"
    "errors"
    "fmt"
    "github.com/bradfitz/gomemcache/memcache"
    "github.com/go-martini/martini"
    "github.com/martini-contrib/binding"
    "github.com/martini-contrib/render"
    "github.com/ncw/swift"
    "log"
    "net/http"
    "os"
    "strings"
    "time"
)

type Paste struct {
    PasteId       string `json:"paste_id"`
    PasteContents string `json:"paste_contents"`
    PasteTtl      string `json:"paste_ttl"`
    PasteType     string `json:"paste_type"`
}

func PanicIf(err error) {
    if err != nil {
        panic(err)
    }
}

func IndexPage(ren render.Render, r *http.Request) {
    ren.HTML(200, "index", nil)
}

func genPasteId() string {
    //first part of a uuid4 is "good enough"
    return strings.SplitN(uuid.New(), "-", 2)[0]
}

func (paste Paste) Validate(errors *binding.Errors, req *http.Request) {
    if paste.PasteContents == "" {
        log.Println("No paste contents")
        errors.Fields["paste_content"] = "Missing"
    }
    if paste.PasteTtl == "" {
        log.Println("No paste ttl")
        errors.Fields["paste_ttl"] = "Missing"
    }
    if paste.PasteType == "" {
        log.Println("No paste type")
        errors.Fields["paste_type"] = "Missing"
    }
}

func getTTL(ttlthing string) (int, error) {
    validTtl := map[string]int{"5 Minutes": 300, "60 Minutes": 3600, "1 Day": 86400, "7 Days": 604800, "30 Days": 2592000, "Forever": 0}
    seconds, ok := validTtl[ttlthing]
    if ok {
        return seconds, nil
    } else {
        return 0, errors.New("Invalid TTL")
    }
}

func GetHistory(ren render.Render, r *http.Request, cf *swift.Connection) {
    opts := swift.ObjectsOpts{}
    opts.Prefix = "cfpaste-"
    opts.Limit = 10
    objects, err := cf.ObjectNames("go-cfpaste", &opts)
    pastes := make([]string, 10)
    for i := range objects {
        object, headers, err := cf.Object("go-cfpaste", objects[i])
        PanicIf(err)
        log.Println(object.Name)
        pastes = append(pastes, headers["X-Object-Meta-Pasteid"])
    }
    PanicIf(err)
    ren.HTML(200, "history", pastes)
    return
}

func GetPaste(params martini.Params, ren render.Render, r *http.Request, cf *swift.Connection, mc *memcache.Client) {
    cachedPaste, err := mc.Get(params["pasteid"])
    format := params["format"]
    if err != nil {
        log.Println(err)
    }
    var paste Paste
    paste.PasteId = params["pasteid"]
    if cachedPaste == nil {
        log.Println("Asking swift for ", params["pasteid"])
        cfPaste, err := cf.ObjectGetBytes("go-cfpaste", params["pasteid"])
        if err != nil {
            if err.Error() == "Object Not Found" {
                ren.HTML(404, "404", paste)
                return
            } else {
                log.Println(err)
                ren.Error(500)
                return
            }
        }
        err = json.Unmarshal(cfPaste, &paste)
        PanicIf(err)
    } else {
        log.Println("Cache hit for ", params["pasteid"])
        err = json.Unmarshal(cachedPaste.Value, &paste)
        PanicIf(err)
    }
    if format == "json" {
        ren.JSON(200, paste)
    } else {
        ren.HTML(200, "paste", paste)
    }
    return
}

func SavePaste(paste Paste, ren render.Render, r *http.Request, cf *swift.Connection, mc *memcache.Client) {
    paste.PasteId = genPasteId()
    payload, _ := json.Marshal(paste)
    seconds, err := getTTL(paste.PasteTtl)
    PanicIf(err)
    headers := swift.Headers{}
    now := time.Now()
    pasteIndex := 9999999999 - now.Unix()
    indexKey := fmt.Sprintf("cfpaste-%d", pasteIndex)
    headers["x-object-meta-pastetype"] = paste.PasteType
    headers["x-object-meta-pasteid"] = paste.PasteId
    headers["x-object-meta-pasteindex"] = fmt.Sprintf("%d", pasteIndex)
    if seconds != 0 {
        headers["x-delete-after"] = fmt.Sprintf("%d", seconds)
    }
    buf := bytes.NewBuffer(payload)
    _, err = cf.ObjectPut("go-cfpaste", paste.PasteId, buf, true, "", "application/json; charset=utf-8", headers)
    PanicIf(err)
    // gholt's listing index hack so that he can spy on pastes
    _, err = cf.ObjectPut("go-cfpaste", indexKey, bytes.NewBuffer([]byte("")), true, "", "application/json; charset=utf-8", headers)
    PanicIf(err)
    mc.Set(&memcache.Item{Key: paste.PasteId, Value: payload})
    ren.JSON(200, map[string]interface{}{"pasteid": paste.PasteId})
}

func main() {
    //lol if you don't already use swiftly
    username := os.Getenv("SWIFTLY_AUTH_USER")
    apikey := os.Getenv("SWIFTLY_AUTH_KEY")
    authurl := os.Getenv("SWIFTLY_AUTH_URL")
    region := os.Getenv("SWIFTLY_REGION")
    snet := os.Getenv("SWIFTLY_SNET")
    internal := false
    if strings.ToLower(snet) == "true" {
        internal = true
    }
    //martini looks for a HOST and PORT env var to determine what to listen on
    m := martini.Classic()

    cf := swift.Connection{
        UserName: username,
        ApiKey:   apikey,
        AuthUrl:  authurl,
        Region:   region,
        Internal: internal,
    }

    err := cf.Authenticate()
    PanicIf(err)
    m.Map(&cf)

    mc := memcache.New("127.0.0.1:11211")
    m.Map(mc)

    m.Use(render.Renderer())

    m.Get("/", IndexPage)
    m.Get("/history", GetHistory)
    m.Get("/:pasteid", GetPaste)
    m.Get("/:pasteid/:format", GetPaste)
    m.Post("/paste", binding.Json(Paste{}), binding.ErrorHandler, SavePaste)
    m.Run()
}
