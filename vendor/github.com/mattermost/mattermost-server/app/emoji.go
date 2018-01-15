// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package app

import (
	"bytes"
	"image"
	"image/draw"
	"image/gif"
	_ "image/jpeg"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"

	l4g "github.com/alecthomas/log4go"

	"image/color/palette"

	"github.com/disintegration/imaging"
	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/utils"
)

const (
	MaxEmojiFileSize = 1000 * 1024 // 1 MB
	MaxEmojiWidth    = 128
	MaxEmojiHeight   = 128
)

func (a *App) CreateEmoji(sessionUserId string, emoji *model.Emoji, multiPartImageData *multipart.Form) (*model.Emoji, *model.AppError) {
	// wipe the emoji id so that existing emojis can't get overwritten
	emoji.Id = ""

	// do our best to validate the emoji before committing anything to the DB so that we don't have to clean up
	// orphaned files left over when validation fails later on
	emoji.PreSave()
	if err := emoji.IsValid(); err != nil {
		return nil, err
	}

	if emoji.CreatorId != sessionUserId {
		return nil, model.NewAppError("createEmoji", "api.emoji.create.other_user.app_error", nil, "", http.StatusForbidden)
	}

	if result := <-a.Srv.Store.Emoji().GetByName(emoji.Name); result.Err == nil && result.Data != nil {
		return nil, model.NewAppError("createEmoji", "api.emoji.create.duplicate.app_error", nil, "", http.StatusBadRequest)
	}

	if imageData := multiPartImageData.File["image"]; len(imageData) == 0 {
		err := model.NewAppError("Context", "api.context.invalid_body_param.app_error", map[string]interface{}{"Name": "createEmoji"}, "", http.StatusBadRequest)
		return nil, err
	} else if err := a.UploadEmojiImage(emoji.Id, imageData[0]); err != nil {
		return nil, err
	}

	if result := <-a.Srv.Store.Emoji().Save(emoji); result.Err != nil {
		return nil, result.Err
	} else {
		message := model.NewWebSocketEvent(model.WEBSOCKET_EVENT_EMOJI_ADDED, "", "", "", nil)
		message.Add("emoji", emoji.ToJson())

		a.Publish(message)
		return result.Data.(*model.Emoji), nil
	}
}

func (a *App) GetEmojiList(page, perPage int) ([]*model.Emoji, *model.AppError) {
	if result := <-a.Srv.Store.Emoji().GetList(page*perPage, perPage); result.Err != nil {
		return nil, result.Err
	} else {
		return result.Data.([]*model.Emoji), nil
	}
}

func (a *App) UploadEmojiImage(id string, imageData *multipart.FileHeader) *model.AppError {
	file, err := imageData.Open()
	if err != nil {
		return model.NewAppError("uploadEmojiImage", "api.emoji.upload.open.app_error", nil, "", http.StatusBadRequest)
	}
	defer file.Close()

	buf := bytes.NewBuffer(nil)
	io.Copy(buf, file)

	// make sure the file is an image and is within the required dimensions
	if config, _, err := image.DecodeConfig(bytes.NewReader(buf.Bytes())); err != nil {
		return model.NewAppError("uploadEmojiImage", "api.emoji.upload.image.app_error", nil, "", http.StatusBadRequest)
	} else if config.Width > MaxEmojiWidth || config.Height > MaxEmojiHeight {
		data := buf.Bytes()
		newbuf := bytes.NewBuffer(nil)
		if info, err := model.GetInfoForBytes(imageData.Filename, data); err != nil {
			return err
		} else if info.MimeType == "image/gif" {
			if gif_data, err := gif.DecodeAll(bytes.NewReader(data)); err != nil {
				return model.NewAppError("uploadEmojiImage", "api.emoji.upload.large_image.gif_decode_error", nil, "", http.StatusBadRequest)
			} else {
				resized_gif := resizeEmojiGif(gif_data)
				if err := gif.EncodeAll(newbuf, resized_gif); err != nil {
					return model.NewAppError("uploadEmojiImage", "api.emoji.upload.large_image.gif_encode_error", nil, "", http.StatusBadRequest)
				}
				if err := a.WriteFile(newbuf.Bytes(), getEmojiImagePath(id)); err != nil {
					return err
				}
			}
		} else {
			if img, _, err := image.Decode(bytes.NewReader(data)); err != nil {
				return model.NewAppError("uploadEmojiImage", "api.emoji.upload.large_image.decode_error", nil, "", http.StatusBadRequest)
			} else {
				resized_image := resizeEmoji(img, config.Width, config.Height)
				if err := png.Encode(newbuf, resized_image); err != nil {
					return model.NewAppError("uploadEmojiImage", "api.emoji.upload.large_image.encode_error", nil, "", http.StatusBadRequest)
				}
				if err := a.WriteFile(newbuf.Bytes(), getEmojiImagePath(id)); err != nil {
					return err
				}
			}
		}
	}

	return a.WriteFile(buf.Bytes(), getEmojiImagePath(id))
}

func (a *App) DeleteEmoji(emoji *model.Emoji) *model.AppError {
	if err := (<-a.Srv.Store.Emoji().Delete(emoji.Id, model.GetMillis())).Err; err != nil {
		return err
	}

	a.deleteEmojiImage(emoji.Id)
	a.deleteReactionsForEmoji(emoji.Name)
	return nil
}

func (a *App) GetEmoji(emojiId string) (*model.Emoji, *model.AppError) {
	if !*a.Config().ServiceSettings.EnableCustomEmoji {
		return nil, model.NewAppError("deleteEmoji", "api.emoji.disabled.app_error", nil, "", http.StatusNotImplemented)
	}

	if len(*a.Config().FileSettings.DriverName) == 0 {
		return nil, model.NewAppError("deleteImage", "api.emoji.storage.app_error", nil, "", http.StatusNotImplemented)
	}

	if result := <-a.Srv.Store.Emoji().Get(emojiId, false); result.Err != nil {
		return nil, result.Err
	} else {
		return result.Data.(*model.Emoji), nil
	}
}

func (a *App) GetEmojiImage(emojiId string) (imageByte []byte, imageType string, err *model.AppError) {
	if result := <-a.Srv.Store.Emoji().Get(emojiId, true); result.Err != nil {
		return nil, "", result.Err
	} else {
		var img []byte

		if data, err := a.ReadFile(getEmojiImagePath(emojiId)); err != nil {
			return nil, "", model.NewAppError("getEmojiImage", "api.emoji.get_image.read.app_error", nil, err.Error(), http.StatusNotFound)
		} else {
			img = data
		}

		_, imageType, err := image.DecodeConfig(bytes.NewReader(img))
		if err != nil {
			return nil, "", model.NewAppError("getEmojiImage", "api.emoji.get_image.decode.app_error", nil, err.Error(), http.StatusInternalServerError)
		}

		return img, imageType, nil
	}
}

func resizeEmojiGif(gifImg *gif.GIF) *gif.GIF {
	// Create a new RGBA image to hold the incremental frames.
	firstFrame := gifImg.Image[0].Bounds()
	b := image.Rect(0, 0, firstFrame.Dx(), firstFrame.Dy())
	img := image.NewRGBA(b)

	resizedImage := image.Image(nil)
	// Resize each frame.
	for index, frame := range gifImg.Image {
		bounds := frame.Bounds()
		draw.Draw(img, bounds, frame, bounds.Min, draw.Over)
		resizedImage = resizeEmoji(img, firstFrame.Dx(), firstFrame.Dy())
		gifImg.Image[index] = imageToPaletted(resizedImage)
	}
	// Set new gif width and height
	gifImg.Config.Width = resizedImage.Bounds().Dx()
	gifImg.Config.Height = resizedImage.Bounds().Dy()
	return gifImg
}

func getEmojiImagePath(id string) string {
	return "emoji/" + id + "/image"
}

func resizeEmoji(img image.Image, width int, height int) image.Image {
	emojiWidth := float64(width)
	emojiHeight := float64(height)

	var emoji image.Image
	if emojiHeight <= MaxEmojiHeight && emojiWidth <= MaxEmojiWidth {
		emoji = img
	} else {
		emoji = imaging.Fit(img, MaxEmojiWidth, MaxEmojiHeight, imaging.Lanczos)
	}
	return emoji
}

func imageToPaletted(img image.Image) *image.Paletted {
	b := img.Bounds()
	pm := image.NewPaletted(b, palette.Plan9)
	draw.FloydSteinberg.Draw(pm, b, img, image.ZP)
	return pm
}

func (a *App) deleteEmojiImage(id string) {
	if err := a.MoveFile(getEmojiImagePath(id), "emoji/"+id+"/image_deleted"); err != nil {
		l4g.Error("Failed to rename image when deleting emoji %v", id)
	}
}

func (a *App) deleteReactionsForEmoji(emojiName string) {
	if result := <-a.Srv.Store.Reaction().DeleteAllWithEmojiName(emojiName); result.Err != nil {
		l4g.Warn(utils.T("api.emoji.delete.delete_reactions.app_error"), emojiName)
		l4g.Warn(result.Err)
	}
}
