package main

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"petbots.fbbdev.it/dotmtxbot/dotmtx"
	"petbots.fbbdev.it/dotmtxbot/log"
)

var imgHost string
var imgServiceAddr string
var gifPath string
var mp4Path string

func init() {
	imgHost = os.Getenv("DOTMTXBOT_IMG_HOST")
	if imgHost == "" {
		imgHost = "localhost:3000"
	}

	imgServiceAddr = os.Getenv("DOTMTXBOT_IMG_SERVICE_ADDR")
	if imgServiceAddr == "" {
		imgServiceAddr = "localhost:3000"
	}

	gifPath = os.Getenv("DOTMTXBOT_GIF_PATH")
	if gifPath == "" {
		gifPath = "/dotmtx.gif"
	}

	mp4Path = os.Getenv("DOTMTXBOT_MP4_PATH")
	if gifPath == "" {
		gifPath = "/dotmtx.mp4"
	}
}

const helpMessage = `Invoke me inline in your chat:
@dotmtxbot Speed Width Blank Text

Speed is the number of characters scrolling out of the display in one second; use a negative value to reverse the scrolling direction.

Width is a number specifying the image width multiplier: when Width is 1, the image has the same width as the text. When 0.5, half the text. When 2, twice the text.

Blank is a number specifying the blank space multiplier: when Blank is 1, the text is followed by a blank space of the same width. When 0.5, half the width and so on.

Text is the text to display. Maximum length is %d characters.

When everything works, I will send you a GIF you can post.
When the parameters are wrong, I will send you nothing.
When the generated GIF is too big, I will send a GIF with an error message.
Sometimes, the GIF won't load even if everything worked. In such cases, try deleting all text and rewrite it. If it still doesn't work, you should try quitting telegram and reopening it or even cleaning Telegram's cache.

PRIVACY NOTICE: your requests will never be stored nor traced back to you in any way by the bot. However, remember that this is a completely public service and you should never send private or personal data to this bot.
The GIFs will be cached by a CDN to speed up delivery. Cached GIFs are only accessible by someone who knows the exact text they contain down to the smallest detail, so if they contain private data they should only be accessible by you. Let us stress again, however, that you should NEVER send private data to this bot. Our CDN, Cloudflare, may of course be able to access the GIFs that are stored in their caches, when required by the law. Here is their privacy policy:

https://www.cloudflare.com/trust-hub/privacy-and-data-protection/

Try invoking me in this chat by writing:

@dotmtxbot 4 1 1 HELLO %s`

func handleStart(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	msg := tgbotapi.NewMessage(
		update.Message.Chat.ID,
		fmt.Sprintf(helpMessage, dotmtx.MaxChars, strings.ToUpper(update.SentFrom().UserName)),
	)

	msg.DisableWebPagePreview = true

	if _, err := bot.Send(msg); err != nil {
		log.ErrorLogger.Print("tgbotapi: ", err)
		log.WarningLogger.Printf("could not send message (update_id=%v, chat_id=%v)", update.UpdateID, msg.ChatID)
	}
}

func sendMessage(bot *tgbotapi.BotAPI, chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	if _, err := bot.Send(msg); err != nil {
		log.ErrorLogger.Print("tgbotapi: ", err)
		log.WarningLogger.Printf("could not send message (chat_id=%v)", chatID)
	}
}

func handleInlineQuery(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	re := regexp.MustCompile(`^\s*(\S+\s+\S+\s+\S+)\s+(.*)$`)
	match := re.FindStringSubmatch(update.InlineQuery.Query)

	// log.InfoLogger.Println(match)

	if match == nil || match[2] == "" {
		return
	}

	var speed float64
	var width float64
	var blank float64

	if _, err := fmt.Sscan(match[1], &speed, &width, &blank); err != nil || width <= 0 || blank < 0 {
		return
	}

	// log.InfoLogger.Println(speed, width, blank)

	text := match[2]
	if len(text) > dotmtx.MaxChars {
		return
	}

	params := url.Values{}
	params.Set("speed", fmt.Sprint(speed))
	params.Set("width", fmt.Sprint(width))
	params.Set("blank", fmt.Sprint(blank))
	params.Set("text", text)

	imgURL := url.URL{
		Scheme:   "https",
		Host:     imgHost,
		Path:     mp4Path,
		RawQuery: params.Encode(),
	}

	imgURLStr := imgURL.String()

	result := tgbotapi.NewInlineQueryResultMPEG4GIF(imgURL.RawQuery, imgURLStr)
	result.ThumbURL = imgURLStr

	// log.Println(gif)

	answer := tgbotapi.InlineConfig{
		InlineQueryID: update.InlineQuery.ID,
		Results:       []interface{}{result},
		CacheTime:     1,
		IsPersonal:    true,
	}

	if _, err := bot.Request(answer); err != nil {
		log.ErrorLogger.Print("tgbotapi: ", err)
		log.WarningLogger.Printf("could not send inline query answer (update_id=%v, query_id=%v)", update.UpdateID, answer.InlineQueryID)
	}
}

func main() {
	tgbotapi.SetLogger(log.InfoLogger)

	bot, err := tgbotapi.NewBotAPI(os.Getenv("DOTMTXBOT_TOKEN"))
	if err != nil {
		log.ErrorLogger.Print("tgbotapi: ", err)
		log.FatalLogger.Fatal("could not start bot")
	}

	bot.Debug = false
	log.InfoLogger.Printf("authorized on account %s", bot.Self.UserName)

	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 60

	updates := bot.GetUpdatesChan(updateConfig)

	// start http server
	go func() {
		http.HandleFunc(gifPath, dotmtx.GifHandler)
		http.HandleFunc(mp4Path, dotmtx.Mp4Handler)

		err := http.ListenAndServe(imgServiceAddr, nil)
		if err != http.ErrServerClosed {
			log.ErrorLogger.Print("http: ", err)
			log.FatalLogger.Fatal("http server stopped")
		}
	}()

	for update := range updates {
		if update.Message != nil && update.Message.IsCommand() {
			switch update.Message.Command() {
			case "start":
				go handleStart(bot, update)
			case "haha":
				go sendMessage(bot, update.Message.Chat.ID, "LOL haha classic")
			default:
				go sendMessage(bot, update.Message.Chat.ID, "I don't know that command")
			}
		} else if update.InlineQuery != nil {
			go handleInlineQuery(bot, update)
		}
	}

	os.Exit(0)
}
