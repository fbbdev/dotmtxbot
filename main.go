package main

import (
	"crypto/md5"
	"errors"
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

var publicHost string
var imgServiceAddr string
var gifPath string
var mp4Path string

func init() {
	publicHost = os.Getenv("DOTMTXBOT_PUBLIC_HOST")
	if publicHost == "" {
		publicHost = "localhost:3000"
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

func sendMessage(bot *tgbotapi.BotAPI, chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	if _, err := bot.Send(msg); err != nil {
		log.ErrorLogger.Print("tgbotapi: ", err)
		log.WarningLogger.Printf("could not send message (chat_id=%v)", chatID)
	}
}

const helpMessage = `Invoke me inline in any chat:
@dotmtxbot [Speed] [Width] [Blank] [Text]

Or in this chat:
/render [Speed] [Width] [Blank] [Text]

[Speed] is the number of characters scrolling out of the display in one second; use a negative value to reverse the scrolling direction.

[Width] is a number specifying the image width multiplier: when Width is 1, the image has the same width as the text. When 0.5, half the text. When 2, twice the text.

[Blank] is a number specifying the blank space multiplier: when Blank is 1, the text is followed by a blank space of the same width. When 0.5, half the width and so on.

[Text] is the text to display. Maximum length is %d characters.

Try invoking me inline in this chat! Go to the chatbar and write:
@dotmtxbot 4 1 1 HELLO %s

When everything works, a GIF will pop up which you can post.
When the parameters are wrong, nothing will pop up.
When the generated GIF is too big, I will send a GIF with an error message.

You can also try sending me this message:
/render 4 1 1 HELLO %s

I will reply with a GIF or a message explaining what went wrong.

PRIVACY NOTICE: your requests will never be stored nor traced back to you in any way by the bot. However, remember that this is a completely public service and you should never send private or personal data to this bot.
The GIFs will be cached by a CDN to speed up delivery. Cached GIFs are only accessible by someone who knows the exact text they contain down to the smallest detail, so if they contain private data they should only be accessible by you. Let us stress again, however, that you should NEVER send private data to this bot. Our CDN, Cloudflare, may of course be able to access the GIFs that are stored in their caches, when required by the law. Here is their privacy policy:

https://www.cloudflare.com/trust-hub/privacy-and-data-protection/`

func handleHelp(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	username := strings.ToUpper(update.SentFrom().UserName)

	msg := tgbotapi.NewMessage(
		update.Message.Chat.ID,
		fmt.Sprintf(helpMessage, dotmtx.MaxChars, username, username),
	)

	msg.DisableWebPagePreview = true

	if _, err := bot.Send(msg); err != nil {
		log.ErrorLogger.Print("tgbotapi: ", err)
		log.WarningLogger.Printf("could not send help message (update_id=%v, chat_id=%v)", update.UpdateID, msg.ChatID)
	}
}

//lint:ignore ST1005 the string must be sent as a chat message
var errNotEnoughParams = errors.New("Some parameters are missing! I need [Speed] [Width] [Blank] [Text]. Try asking for /help if you don't know how to invoke me.")

//lint:ignore ST1005 the string must be sent as a chat message
var errInvalidParams = errors.New("Some parameters are not valid. [Speed], [Width] and [Blank] must be numbers. [Width] must not be negative and [Blank] must be greater than zero.")

//lint:ignore ST1005 the string must be sent as a chat message
var errTextTooLong = fmt.Errorf("[Text] is too long. The limit is %v characters.", dotmtx.MaxChars)

func queryToURL(query string) (imgURL string, err error) {
	re := regexp.MustCompile(`^\s*(\S+\s+\S+\s+\S+)\s+(.+)$`)
	match := re.FindStringSubmatch(query)

	// log.InfoLogger.Print("query string matched: ", match)

	if match == nil || match[2] == "" {
		return "", errNotEnoughParams
	}

	var speed float64
	var width float64
	var blank float64

	if _, ierr := fmt.Sscan(match[1], &speed, &width, &blank); ierr != nil || width <= 0 || blank < 0 {
		return "", errInvalidParams
	}

	// log.InfoLogger.Print("speed=", speed, ", width=", width, ", blank=", blank)

	text := match[2]
	if len(text) > dotmtx.MaxChars {
		return "", errTextTooLong
	}

	params := url.Values{}
	params.Set("speed", fmt.Sprint(speed))
	params.Set("width", fmt.Sprint(width))
	params.Set("blank", fmt.Sprint(blank))
	params.Set("text", text)

	imgURLInfo := url.URL{
		Scheme:   "https",
		Host:     publicHost,
		Path:     mp4Path,
		RawQuery: params.Encode(),
	}

	return imgURLInfo.String(), nil
}

func handleRender(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	query := update.Message.CommandArguments()
	if query == "" {
		sendMessage(bot, update.Message.Chat.ID, "Some parameters are missing:\n/render [Speed] [Width] [Blank] [Text]\n\nJust ask if you need some /help")
		return
	}

	imgURL, err := queryToURL(query)
	if err != nil {
		sendMessage(bot, update.Message.Chat.ID, err.Error())
		return
	}

	msg := tgbotapi.NewAnimation(update.Message.Chat.ID, tgbotapi.FileURL(imgURL))

	if _, err := bot.Send(msg); err != nil {
		log.ErrorLogger.Print("tgbotapi: ", err)
		log.WarningLogger.Printf("could not send rendered GIF (update_id=%v, chat_id=%v)", update.UpdateID, msg.ChatID)
	}
}

func handleInlineQuery(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	imgURL, err := queryToURL(update.InlineQuery.Query)
	if err != nil {
		return
	}

	result := tgbotapi.NewInlineQueryResultMPEG4GIF(fmt.Sprintf("%x", md5.Sum([]byte(imgURL))), imgURL)
	result.ThumbURL = imgURL

	// log.InfoLogger.Print(result)

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
			case "start", "help":
				go handleHelp(bot, update)
			case "render":
				go handleRender(bot, update)
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
