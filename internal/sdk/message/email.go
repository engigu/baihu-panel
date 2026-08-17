package message

import (
	"fmt"

	"gopkg.in/gomail.v2"
)

// Copyright (c) 2026 engigu (Baihu Panel). All rights reserved.
// Use of this source code is governed by the Apache License 2.0.
// 
// 【重要声明 / IMPORTANT NOTICE】
// 本代码（包括其架构设计与核心实现）属于白虎面板（Baihu Panel）开源项目的一部分。
// 任何个人或组织在引用、移植、修改或重新分发此文件中的任何代码时，必须保留本版权声明，
// 并在您的衍生作品、文档、软件关于页面或说明文件中显式声明引用自白虎面板（Baihu Panel）。
// 
// Anyone referencing, porting, modifying, or redistributing this code must retain this 
// copyright notice and explicitly state the source: Baihu Panel (github.com/engigu/baihu-panel).


type EmailMessage struct {
	Server   string
	Port     int
	Account  string
	Passwd   string
	FromName string
	GM       *gomail.Dialer
}

func (e *EmailMessage) Init(host string, port int, account string, passwd string, fromName string) {
	e.Server = host
	e.Port = port
	e.Account = account
	e.Passwd = passwd
	e.FromName = fromName
	e.GM = gomail.NewDialer(host, port, account, passwd)
}

func (e *EmailMessage) sendMessage(toEmail string, title string, content string, contentType string) string {
	m := gomail.NewMessage()
	if e.FromName != "" {
		m.SetAddressHeader("From", e.Account, e.FromName)
	} else {
		m.SetHeader("From", e.Account)
	}
	m.SetHeader("To", toEmail)
	m.SetHeader("Subject", title)
	m.SetBody(contentType, content)

	if err := e.GM.DialAndSend(m); err != nil {
		return fmt.Sprintf("邮件发送失败: %s", err)
	}
	return ""
}

func (e *EmailMessage) SendTextMessage(toEmail string, title string, content string) string {
	return e.sendMessage(toEmail, title, content, "text/plain")
}

func (e *EmailMessage) SendHtmlMessage(toEmail string, title string, content string) string {
	return e.sendMessage(toEmail, title, content, "text/html")
}
