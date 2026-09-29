package main

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"net/http"
	"os"
	"strings"
	"time"
)

type Config struct {
	Port              string `json:"port"`
	GenerateEndpoint  string `json:"generate_endpoint"`
	DetectEndpoint    string `json:"detect_endpoint"`
	AskEndpoint       string `json:"ask_endpoint"`
	EnableLandingPage bool   `json:"enable_landing_page"`
	BarkLikeEndpoint  string `json:"barklike_endpoint"`
}

func loadConfig() Config {
	cfg := Config{
		Port:              "8000",
		GenerateEndpoint:  "/bark",
		DetectEndpoint:    "/isbark",
		AskEndpoint:       "/askbark",
		BarkLikeEndpoint:  "/barklike",
		EnableLandingPage: true,
	}

	file, err := os.Open("config.json")
	if err == nil {
		defer file.Close()
		json.NewDecoder(file).Decode(&cfg)
		fmt.Println("Config loaded successfully 🌸ヽ(^。^)丿")
	} else {
		fmt.Println("Config file not found, using defaults woof~")
	}

	return cfg
}

func main() {
	cfg := loadConfig()

	if cfg.EnableLandingPage {
		http.HandleFunc("/", rootHandler)
		fmt.Println("   -> Landing Page:   Enabled")
	} else {
		fmt.Println("   -> Landing Page:   Disabled")
	}

	http.HandleFunc(cfg.GenerateEndpoint, generateBark)
	http.HandleFunc(cfg.DetectEndpoint, detectBark)
	http.HandleFunc(cfg.AskEndpoint, askBark)
	http.HandleFunc(cfg.BarkLikeEndpoint, detectBarkLike)
	http.HandleFunc("/languages", languagesHandler)

	fmt.Printf("BaaS is running on port %s (ᐢ ᴥ ᐢ)\n", cfg.Port)
	fmt.Printf("    -> Generation: %s\n", cfg.GenerateEndpoint)
	fmt.Printf("    -> Detection: %s\n", cfg.DetectEndpoint)
	fmt.Printf("    -> 8-Ball:     %s\n", cfg.AskEndpoint)
	fmt.Printf("    -> Bark-Like:  %s\n", cfg.BarkLikeEndpoint)

	http.ListenAndServe(":"+cfg.Port, nil)
}

var askBarkAnswers = []string{
	"It is certain. Woof.",
	"The tail says yes.",
	"Without a doubt. Woof woof.",
	"Yes, absolutely.",
	"You may rely on woof.",
	"As I see it, yes. Woof.",
	"Most likely. Tail wagging.",
	"Outlook good. Woof.",
	"Yes, and there's a treat in it for you.",
	"All paws point to yes.",
	"Absolutely. Woof!",
	"The pack agrees: yes.",
	"The wagging tail agrees. Woof.",
	"Paws down, yes.",
	"Certified good idea. Woof.",
	"The answer is yes, even before the walk.",
	"The council of dogs has spoken: woof.",
	"That's a yes from this dog.",
	"Yes beyond reasonable doubt. Woof.",
	"Verdict: yes. Fetch it.",
	"Probably yes. Woof.",
	"Looks good from here.",
	"I'd bet a treat on yes.",
	"The tail twitch says yes.",
	"A soft woof of approval.",
	"Favorable conditions for a walk and a yes.",
	"Yes, with a little cautious sniffing.",
	"Soft woof, strong yes.",
	"The dog park approves.",
	"A yes is on its way. Woof.",
	"Reply hazy. Bark again in a minute.",
	"Ask again after the walk.",
	"The ears are up, but the answer is unclear.",
	"Cannot predict yet. Woof?",
	"Sniff around and ask again.",
	"The dog park is undecided.",
	"Woof lost in translation.",
	"The leash says both yes and no.",
	"Flip a biscuit and ask again.",
	"The dog walked across the keyboard. Ask later.",
	"Insufficient treats for a clear answer.",
	"Woof signal weak. Move closer to the food bowl.",
	"Nap first, answer later.",
	"The tennis ball says: unclear.",
	"Static on the woof line.",
	"Ask after the zoomies.",
	"Results may vary by dog mood.",
	"The answer is still sniffing around.",
	"The window stare continues. No answer yet.",
	"Don't count on it. Woof.",
	"My reply is no.",
	"My sources say no. Woof.",
	"Outlook not so good.",
	"Very doubtful. Ears down.",
	"Hard no. Woof.",
	"Not this time, friend.",
	"The paws say no.",
	"Request denied. Woof.",
	"A firm no from the pack.",
	"Only silence from the dog bed.",
	"The answer has been vetoed.",
	"The empty bowl disagrees.",
	"Unlikely. Woof.",
	"In this economy? No woof.",
	"Probably not, but I admire the hope.",
	"Unlikely, but admire the audacity.",
	"The tail isn't wagging.",
	"The tail is low on this one.",
	"Not looking good, pup.",
	"Forecast: cloudy with a chance of no.",
	"Have you tried barking at it?",
	"Woof is a state of mind.",
	"Only on a very good dog day.",
	"Ask the dog by the door.",
	"Bark once for yes, twice for also yes.",
	"The tennis ball knows, but won't tell.",
	"Woof is usually the answer. This is the exception.",
	"Woof louder.",
	"Bark responsibly.",
	"According to ancient dog law: woof.",
	"A woof has been detected. Meaning unclear.",
	"That's between you and the dog park.",
	"Woof at home, howl at the moon.",
	"Error 404: woof not found.",
	"The chew toy has no comment.",
	"Bark loudly and carry a big stick.",
	"Yes, but make it fashion. Woof.",
	"No, but have you considered a nap?",
	"Woof is not a number. Wait, yes it is.",
	"The dog distribution system has allocated: one woof.",
	"Woof approved by 9 out of 10 dogs. The tenth is asleep.",
	"Biologically speaking, woof.",
	"Philosophically, what is woof?",
	"Legally distinct from woof, but spiritually woof.",
	"Woof, but whisper it.",
	"WOOF.",
	"The prophecy foretold woof.",
	"Ask not whether it is woof, but whether you are worthy of woof.",
}

func askBark(w http.ResponseWriter, r *http.Request) {
	if !requireGET(w, r) {
		return
	}
	start := time.Now()
	values, queryErr := parseRequestQuery(r)
	if queryErr != nil {
		writeAPIError(w, queryErr)
		return
	}
	if _, supplied := values["lang"]; supplied {
		writeAPIJSON(w, http.StatusBadRequest, map[string]string{"error": "lang is not supported on /askbark"})
		return
	}
	text, apiErr := getRequestText(r, "a question", "is it time for food")
	if apiErr != nil {
		writeAPIError(w, apiErr)
		return
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(strings.ToLower(strings.TrimSpace(text))))
	answer := askBarkAnswers[h.Sum32()%uint32(len(askBarkAnswers))]
	writeAPIJSON(w, http.StatusOK, map[string]interface{}{
		"question": text, "answer": answer,
		"latency_time": time.Since(start).String(),
	})
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	html := `<!DOCTYPE html>
<html>
<head>
    <title>BaaS API</title>
    <style>
        body { font-family: monospace; background: #111; color: #eee; padding: 2rem; max-width: 600px; margin: auto; }
        a { color: #ffd1dc; text-decoration: none; }
        a:hover { text-decoration: underline; }
        ul { list-style-type: none; padding-left: 0; }
        li { margin-bottom: 1rem; background: #222; padding: 1rem; border-radius: 4px; }
    </style>
</head>
<body>
    <p>woof.</p>
    <p>contribute: <a href="https://github.com/7w1/Bark">github.com/7w1/Bark</a></p>
    <br>
    <h3>Endpoints</h3>
	<p>There is a per-ip rate limit of 100 requests per 10 seconds.</p>
    <ul>
        <li>
            <code>GET <a href="/bark">/bark</a></code><br><br>
            Generates a vocalization. Add a supported <code>?lang=...</code> tag to choose its language.
        </li>
        <li>
            <code>GET <a href="/isbark?text=woof">/isbark?text={input}</a></code><br><br>
            Strictly checks the complete input in the selected language (default: English). Example: <code>/isbark?text=ワンワン&amp;lang=ja</code>.
        </li>
		<li>
            <code>GET <a href="/barklike?text=waf">/barklike?text={input}</a></code><br><br>
            Checks the complete input and substantial word-level vocalizations. An embedded result reports Unicode-letter coverage; this score is not a probability.
        </li>
        <li>
            <code>GET <a href="/askbark?text=hello">/askbark?text={question}</a></code><br><br>
            Returns a deterministic English response.
        </li>
    </ul>
    <p>Supported bark languages: <code>en</code>, <code>fr</code>, <code>ja</code>, <code>tr</code>, <code>ru</code>, <code>uk</code>, <code>zh-Hans</code>, <code>zh-Hant</code>, <code>es</code>, <code>nl</code>, <code>de</code>, <code>hy</code>, <code>eo</code>, <code>ar</code>, <code>hi</code>. Detection accepts <code>lang=auto</code>. Use <a href="/languages">/languages</a> for the current list.</p>
    <br>
    <p>Made by 7w1</p>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}
