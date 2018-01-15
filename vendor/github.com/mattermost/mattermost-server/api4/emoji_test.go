// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package api4

import (
	"bytes"
	"image"
	_ "image/gif"
	"testing"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/utils"
)

func TestCreateEmoji(t *testing.T) {
	th := Setup().InitBasic().InitSystemAdmin()
	defer th.TearDown()
	Client := th.Client

	EnableCustomEmoji := *th.App.Config().ServiceSettings.EnableCustomEmoji
	defer func() {
		th.App.UpdateConfig(func(cfg *model.Config) { *cfg.ServiceSettings.EnableCustomEmoji = EnableCustomEmoji })
	}()
	th.App.UpdateConfig(func(cfg *model.Config) { *cfg.ServiceSettings.EnableCustomEmoji = false })

	emoji := &model.Emoji{
		CreatorId: th.BasicUser.Id,
		Name:      model.NewId(),
	}

	// try to create an emoji when they're disabled
	_, resp := Client.CreateEmoji(emoji, utils.CreateTestGif(t, 10, 10), "image.gif")
	CheckNotImplementedStatus(t, resp)

	th.App.UpdateConfig(func(cfg *model.Config) { *cfg.ServiceSettings.EnableCustomEmoji = true })
	// try to create a valid gif emoji when they're enabled
	newEmoji, resp := Client.CreateEmoji(emoji, utils.CreateTestGif(t, 10, 10), "image.gif")
	CheckNoError(t, resp)
	if newEmoji.Name != emoji.Name {
		t.Fatal("create with wrong name")
	}

	// try to create an emoji with a duplicate name
	emoji2 := &model.Emoji{
		CreatorId: th.BasicUser.Id,
		Name:      newEmoji.Name,
	}
	_, resp = Client.CreateEmoji(emoji2, utils.CreateTestGif(t, 10, 10), "image.gif")
	CheckBadRequestStatus(t, resp)
	CheckErrorMessage(t, resp, "api.emoji.create.duplicate.app_error")

	// try to create a valid animated gif emoji
	emoji = &model.Emoji{
		CreatorId: th.BasicUser.Id,
		Name:      model.NewId(),
	}

	newEmoji, resp = Client.CreateEmoji(emoji, utils.CreateTestAnimatedGif(t, 10, 10, 10), "image.gif")
	CheckNoError(t, resp)
	if newEmoji.Name != emoji.Name {
		t.Fatal("create with wrong name")
	}

	// try to create a valid jpeg emoji
	emoji = &model.Emoji{
		CreatorId: th.BasicUser.Id,
		Name:      model.NewId(),
	}

	newEmoji, resp = Client.CreateEmoji(emoji, utils.CreateTestJpeg(t, 10, 10), "image.gif")
	CheckNoError(t, resp)
	if newEmoji.Name != emoji.Name {
		t.Fatal("create with wrong name")
	}

	// try to create a valid png emoji
	emoji = &model.Emoji{
		CreatorId: th.BasicUser.Id,
		Name:      model.NewId(),
	}

	newEmoji, resp = Client.CreateEmoji(emoji, utils.CreateTestPng(t, 10, 10), "image.gif")
	CheckNoError(t, resp)
	if newEmoji.Name != emoji.Name {
		t.Fatal("create with wrong name")
	}

	// try to create an emoji that's too wide
	emoji = &model.Emoji{
		CreatorId: th.BasicUser.Id,
		Name:      model.NewId(),
	}

	newEmoji, resp = Client.CreateEmoji(emoji, utils.CreateTestGif(t, 1000, 10), "image.gif")
	CheckNoError(t, resp)
	if newEmoji.Name != emoji.Name {
		t.Fatal("create with wrong name")
	}

	// try to create an emoji that's too tall
	emoji = &model.Emoji{
		CreatorId: th.BasicUser.Id,
		Name:      model.NewId(),
	}

	newEmoji, resp = Client.CreateEmoji(emoji, utils.CreateTestGif(t, 10, 1000), "image.gif")
	CheckNoError(t, resp)
	if newEmoji.Name != emoji.Name {
		t.Fatal("create with wrong name")
	}

	// try to create an emoji that's too large
	emoji = &model.Emoji{
		CreatorId: th.BasicUser.Id,
		Name:      model.NewId(),
	}

	_, resp = Client.CreateEmoji(emoji, utils.CreateTestAnimatedGif(t, 100, 100, 10000), "image.gif")
	if resp.Error == nil {
		t.Fatal("should fail - emoji is too big")
	}

	// try to create an emoji with data that isn't an image
	emoji = &model.Emoji{
		CreatorId: th.BasicUser.Id,
		Name:      model.NewId(),
	}

	_, resp = Client.CreateEmoji(emoji, make([]byte, 100), "image.gif")
	CheckBadRequestStatus(t, resp)
	CheckErrorMessage(t, resp, "api.emoji.upload.image.app_error")

	// try to create an emoji as another user
	emoji = &model.Emoji{
		CreatorId: th.BasicUser2.Id,
		Name:      model.NewId(),
	}

	_, resp = Client.CreateEmoji(emoji, utils.CreateTestGif(t, 10, 10), "image.gif")
	CheckForbiddenStatus(t, resp)
}

func TestGetEmojiList(t *testing.T) {
	th := Setup().InitBasic()
	defer th.TearDown()
	Client := th.Client

	EnableCustomEmoji := *th.App.Config().ServiceSettings.EnableCustomEmoji
	defer func() {
		th.App.UpdateConfig(func(cfg *model.Config) { *cfg.ServiceSettings.EnableCustomEmoji = EnableCustomEmoji })
	}()
	th.App.UpdateConfig(func(cfg *model.Config) { *cfg.ServiceSettings.EnableCustomEmoji = true })

	emojis := []*model.Emoji{
		{
			CreatorId: th.BasicUser.Id,
			Name:      model.NewId(),
		},
		{
			CreatorId: th.BasicUser.Id,
			Name:      model.NewId(),
		},
		{
			CreatorId: th.BasicUser.Id,
			Name:      model.NewId(),
		},
	}

	for idx, emoji := range emojis {
		emoji, resp := Client.CreateEmoji(emoji, utils.CreateTestGif(t, 10, 10), "image.gif")
		CheckNoError(t, resp)
		emojis[idx] = emoji
	}

	listEmoji, resp := Client.GetEmojiList(0, 100)
	CheckNoError(t, resp)
	for _, emoji := range emojis {
		found := false
		for _, savedEmoji := range listEmoji {
			if emoji.Id == savedEmoji.Id {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("failed to get emoji with id %v", emoji.Id)
		}
	}

	_, resp = Client.DeleteEmoji(emojis[0].Id)
	CheckNoError(t, resp)
	listEmoji, resp = Client.GetEmojiList(0, 100)
	CheckNoError(t, resp)
	found := false
	for _, savedEmoji := range listEmoji {
		if savedEmoji.Id == emojis[0].Id {
			found = true
			break
		}
		if found {
			t.Fatalf("should not get a deleted emoji %v", emojis[0].Id)
		}
	}

	listEmoji, resp = Client.GetEmojiList(0, 1)
	CheckNoError(t, resp)

	if len(listEmoji) != 1 {
		t.Fatal("should only return 1")
	}
}

func TestDeleteEmoji(t *testing.T) {
	th := Setup().InitBasic().InitSystemAdmin()
	defer th.TearDown()
	Client := th.Client

	EnableCustomEmoji := *th.App.Config().ServiceSettings.EnableCustomEmoji
	defer func() {
		th.App.UpdateConfig(func(cfg *model.Config) { *cfg.ServiceSettings.EnableCustomEmoji = EnableCustomEmoji })
	}()
	th.App.UpdateConfig(func(cfg *model.Config) { *cfg.ServiceSettings.EnableCustomEmoji = true })

	emoji := &model.Emoji{
		CreatorId: th.BasicUser.Id,
		Name:      model.NewId(),
	}

	newEmoji, resp := Client.CreateEmoji(emoji, utils.CreateTestGif(t, 10, 10), "image.gif")
	CheckNoError(t, resp)

	ok, resp := Client.DeleteEmoji(newEmoji.Id)
	CheckNoError(t, resp)
	if !ok {
		t.Fatal("should return true")
	} else {
		_, err := Client.GetEmoji(newEmoji.Id)
		if err == nil {
			t.Fatal("should not return the emoji it was deleted")
		}
	}

	//Admin can delete other users emoji
	newEmoji, resp = Client.CreateEmoji(emoji, utils.CreateTestGif(t, 10, 10), "image.gif")
	CheckNoError(t, resp)

	ok, resp = th.SystemAdminClient.DeleteEmoji(newEmoji.Id)
	CheckNoError(t, resp)
	if !ok {
		t.Fatal("should return true")
	} else {
		_, err := th.SystemAdminClient.GetEmoji(newEmoji.Id)
		if err == nil {
			t.Fatal("should not return the emoji it was deleted")
		}
	}

	// Try to delete just deleted emoji
	_, resp = Client.DeleteEmoji(newEmoji.Id)
	CheckNotFoundStatus(t, resp)

	//Try to delete non-existing emoji
	_, resp = Client.DeleteEmoji(model.NewId())
	CheckNotFoundStatus(t, resp)

	//Try to delete without Id
	_, resp = Client.DeleteEmoji("")
	CheckNotFoundStatus(t, resp)

	//Try to delete other user's custom emoji
	newEmoji, resp = Client.CreateEmoji(emoji, utils.CreateTestGif(t, 10, 10), "image.gif")
	CheckNoError(t, resp)

	Client.Logout()
	th.LoginBasic2()
	ok, resp = Client.DeleteEmoji(newEmoji.Id)
	CheckUnauthorizedStatus(t, resp)
}

func TestGetEmoji(t *testing.T) {
	th := Setup().InitBasic()
	defer th.TearDown()
	Client := th.Client

	EnableCustomEmoji := *th.App.Config().ServiceSettings.EnableCustomEmoji
	defer func() {
		th.App.UpdateConfig(func(cfg *model.Config) { *cfg.ServiceSettings.EnableCustomEmoji = EnableCustomEmoji })
	}()
	th.App.UpdateConfig(func(cfg *model.Config) { *cfg.ServiceSettings.EnableCustomEmoji = true })

	emoji := &model.Emoji{
		CreatorId: th.BasicUser.Id,
		Name:      model.NewId(),
	}

	newEmoji, resp := Client.CreateEmoji(emoji, utils.CreateTestGif(t, 10, 10), "image.gif")
	CheckNoError(t, resp)

	emoji, resp = Client.GetEmoji(newEmoji.Id)
	CheckNoError(t, resp)
	if emoji.Id != newEmoji.Id {
		t.Fatal("wrong emoji was returned")
	}

	_, resp = Client.GetEmoji(model.NewId())
	CheckNotFoundStatus(t, resp)
}

func TestGetEmojiImage(t *testing.T) {
	th := Setup().InitBasic()
	defer th.TearDown()
	Client := th.Client

	EnableCustomEmoji := *th.App.Config().ServiceSettings.EnableCustomEmoji
	DriverName := *th.App.Config().FileSettings.DriverName
	defer func() {
		th.App.UpdateConfig(func(cfg *model.Config) { *cfg.ServiceSettings.EnableCustomEmoji = EnableCustomEmoji })
		th.App.UpdateConfig(func(cfg *model.Config) { *cfg.FileSettings.DriverName = DriverName })
	}()
	th.App.UpdateConfig(func(cfg *model.Config) { *cfg.ServiceSettings.EnableCustomEmoji = true })

	emoji1 := &model.Emoji{
		CreatorId: th.BasicUser.Id,
		Name:      model.NewId(),
	}

	emoji1, resp := Client.CreateEmoji(emoji1, utils.CreateTestGif(t, 10, 10), "image.gif")
	CheckNoError(t, resp)

	th.App.UpdateConfig(func(cfg *model.Config) { *cfg.ServiceSettings.EnableCustomEmoji = false })

	_, resp = Client.GetEmojiImage(emoji1.Id)
	CheckNotImplementedStatus(t, resp)
	CheckErrorMessage(t, resp, "api.emoji.disabled.app_error")

	th.App.UpdateConfig(func(cfg *model.Config) { *cfg.FileSettings.DriverName = "" })
	th.App.UpdateConfig(func(cfg *model.Config) { *cfg.ServiceSettings.EnableCustomEmoji = true })

	_, resp = Client.GetEmojiImage(emoji1.Id)
	CheckNotImplementedStatus(t, resp)
	CheckErrorMessage(t, resp, "api.emoji.storage.app_error")

	th.App.UpdateConfig(func(cfg *model.Config) { *cfg.FileSettings.DriverName = DriverName })

	emojiImage, resp := Client.GetEmojiImage(emoji1.Id)
	CheckNoError(t, resp)
	if len(emojiImage) <= 0 {
		t.Fatal("should return the image")
	}
	_, imageType, err := image.DecodeConfig(bytes.NewReader(emojiImage))
	if err != nil {
		t.Fatalf("unable to identify received image: %v", err.Error())
	} else if imageType != "gif" {
		t.Fatal("should've received gif data")
	}

	emoji2 := &model.Emoji{
		CreatorId: th.BasicUser.Id,
		Name:      model.NewId(),
	}

	emoji2, resp = Client.CreateEmoji(emoji2, utils.CreateTestAnimatedGif(t, 10, 10, 10), "image.gif")
	CheckNoError(t, resp)

	emojiImage, resp = Client.GetEmojiImage(emoji2.Id)
	CheckNoError(t, resp)
	if len(emojiImage) <= 0 {
		t.Fatal("should return the image")
	}
	_, imageType, err = image.DecodeConfig(bytes.NewReader(emojiImage))
	if err != nil {
		t.Fatalf("unable to identify received image: %v", err.Error())
	} else if imageType != "gif" {
		t.Fatal("should've received gif data")
	}

	emoji3 := &model.Emoji{
		CreatorId: th.BasicUser.Id,
		Name:      model.NewId(),
	}
	emoji3, resp = Client.CreateEmoji(emoji3, utils.CreateTestJpeg(t, 10, 10), "image.jpg")
	CheckNoError(t, resp)

	emojiImage, resp = Client.GetEmojiImage(emoji3.Id)
	CheckNoError(t, resp)
	if len(emojiImage) <= 0 {
		t.Fatal("should return the image")
	}
	_, imageType, err = image.DecodeConfig(bytes.NewReader(emojiImage))
	if err != nil {
		t.Fatalf("unable to identify received image: %v", err.Error())
	} else if imageType != "jpeg" {
		t.Fatal("should've received gif data")
	}

	emoji4 := &model.Emoji{
		CreatorId: th.BasicUser.Id,
		Name:      model.NewId(),
	}
	emoji4, resp = Client.CreateEmoji(emoji4, utils.CreateTestPng(t, 10, 10), "image.png")
	CheckNoError(t, resp)

	emojiImage, resp = Client.GetEmojiImage(emoji4.Id)
	CheckNoError(t, resp)
	if len(emojiImage) <= 0 {
		t.Fatal("should return the image")
	}
	_, imageType, err = image.DecodeConfig(bytes.NewReader(emojiImage))
	if err != nil {
		t.Fatalf("unable to identify received image: %v", err.Error())
	} else if imageType != "png" {
		t.Fatal("should've received gif data")
	}

	_, resp = Client.DeleteEmoji(emoji4.Id)
	CheckNoError(t, resp)

	_, resp = Client.GetEmojiImage(emoji4.Id)
	CheckNotFoundStatus(t, resp)

	_, resp = Client.GetEmojiImage(model.NewId())
	CheckNotFoundStatus(t, resp)

	_, resp = Client.GetEmojiImage("")
	CheckBadRequestStatus(t, resp)
}
