package reports

import "errors"

// KHUNG TẠM của commit đỏ-trước (bước 3, quyết định của người và bản rời nhóm). Bị thay bằng cài đặt thật ở commit kế tiếp.

func (c *opctx) decide(cmd Command) error                           { return errors.New("chưa cài") }
func (c *opctx) retract(key, actor, why string) error               { return errors.New("chưa cài") }
func (c *opctx) amend(key string, it Item, actor, why string) error { return errors.New("chưa cài") }
