package server

import (
	"github.com/emersion/go-smtp"
	"github.com/stretchr/testify/require"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
)

func TestSmtpBackend_Multipart(t *testing.T) {
	email := `MIME-Version: 1.0
Date: Tue, 28 Dec 2021 00:30:10 +0100
Message-ID: <CAAvm79YP0C=Rt1N=KWmSUBB87KK2rRChmdzKqF1vCwMEUiVzLQ@mail.gmail.com>
Subject: and one more
From: Phil <phil@example.com>
To: ntfy-mytopic@ntfy.sh
Content-Type: multipart/alternative; boundary="000000000000f3320b05d42915c9"

--000000000000f3320b05d42915c9
Content-Type: text/plain; charset="UTF-8"

what's up

--000000000000f3320b05d42915c9
Content-Type: text/html; charset="UTF-8"

<div dir="ltr">what&#39;s up<br clear="all"><div><br></div></div>

--000000000000f3320b05d42915c9--`
	_, backend := newTestBackend(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/mytopic", r.URL.Path)
		require.Equal(t, "and one more", r.Header.Get("Title"))
		require.Equal(t, "what's up", readAll(t, r.Body))
	})
	session, _ := backend.AnonymousLogin(fakeConnState(t, "1.2.3.4"))
	require.Nil(t, session.Mail("phil@example.com", smtp.MailOptions{}))
	require.Nil(t, session.Rcpt("ntfy-mytopic@ntfy.sh"))
	require.Nil(t, session.Data(strings.NewReader(email)))
}

func TestSmtpBackend_MultipartNoBody(t *testing.T) {
	email := `MIME-Version: 1.0
Date: Tue, 28 Dec 2021 01:33:34 +0100
Message-ID: <CAAvm7ABCDsi9vsuu0WTRXzZQBC8dXrDOLT8iCWdqrsmg@mail.gmail.com>
Subject: This email has a subject but no body
From: Phil <phil@example.com>
To: ntfy-emailtest@ntfy.sh
Content-Type: multipart/alternative; boundary="000000000000bcf4a405d429f8d4"

--000000000000bcf4a405d429f8d4
Content-Type: text/plain; charset="UTF-8"



--000000000000bcf4a405d429f8d4
Content-Type: text/html; charset="UTF-8"

<div dir="ltr"><br></div>

--000000000000bcf4a405d429f8d4--`
	_, backend := newTestBackend(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/emailtest", r.URL.Path)
		require.Equal(t, "", r.Header.Get("Title")) // We flipped message and body
		require.Equal(t, "This email has a subject but no body", readAll(t, r.Body))
	})
	session, _ := backend.AnonymousLogin(fakeConnState(t, "1.2.3.4"))
	require.Nil(t, session.Mail("phil@example.com", smtp.MailOptions{}))
	require.Nil(t, session.Rcpt("ntfy-emailtest@ntfy.sh"))
	require.Nil(t, session.Data(strings.NewReader(email)))
}

func TestSmtpBackend_Plaintext(t *testing.T) {
	email := `Date: Tue, 28 Dec 2021 00:30:10 +0100
Message-ID: <CAAvm79YP0C=Rt1N=KWmSUBB87KK2rRChmdzKqF1vCwMEUiVzLQ@mail.gmail.com>
Subject: and one more
From: Phil <phil@example.com>
To: mytopic@ntfy.sh
Content-Type: text/plain; charset="UTF-8"

what's up
`
	conf, backend := newTestBackend(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/mytopic", r.URL.Path)
		require.Equal(t, "and one more", r.Header.Get("Title"))
		require.Equal(t, "what's up", readAll(t, r.Body))
	})
	conf.SMTPServerAddrPrefix = ""
	session, _ := backend.AnonymousLogin(fakeConnState(t, "1.2.3.4"))
	require.Nil(t, session.Mail("phil@example.com", smtp.MailOptions{}))
	require.Nil(t, session.Rcpt("mytopic@ntfy.sh"))
	require.Nil(t, session.Data(strings.NewReader(email)))
}

func TestSmtpBackend_Plaintext_No_ContentType(t *testing.T) {
	email := `Subject: Very short mail

what's up
`
	conf, backend := newTestBackend(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/mytopic", r.URL.Path)
		require.Equal(t, "Very short mail", r.Header.Get("Title"))
		require.Equal(t, "what's up", readAll(t, r.Body))
	})
	conf.SMTPServerAddrPrefix = ""
	session, _ := backend.AnonymousLogin(fakeConnState(t, "1.2.3.4"))
	require.Nil(t, session.Mail("phil@example.com", smtp.MailOptions{}))
	require.Nil(t, session.Rcpt("mytopic@ntfy.sh"))
	require.Nil(t, session.Data(strings.NewReader(email)))
}

func TestSmtpBackend_Plaintext_EncodedSubject(t *testing.T) {
	email := `Date: Tue, 28 Dec 2021 00:30:10 +0100
Subject: =?UTF-8?B?VGhyZWUgc2FudGFzIPCfjoXwn46F8J+OhQ==?=
From: Phil <phil@example.com>
To: ntfy-mytopic@ntfy.sh
Content-Type: text/plain; charset="UTF-8"

what's up
`
	_, backend := newTestBackend(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "Three santas 🎅🎅🎅", r.Header.Get("Title"))
	})
	session, _ := backend.AnonymousLogin(fakeConnState(t, "1.2.3.4"))
	require.Nil(t, session.Mail("phil@example.com", smtp.MailOptions{}))
	require.Nil(t, session.Rcpt("ntfy-mytopic@ntfy.sh"))
	require.Nil(t, session.Data(strings.NewReader(email)))
}

func TestSmtpBackend_Plaintext_TooLongTruncate(t *testing.T) {
	email := `Date: Tue, 28 Dec 2021 00:30:10 +0100
Message-ID: <CAAvm79YP0C=Rt1N=KWmSUBB87KK2rRChmdzKqF1vCwMEUiVzLQ@mail.gmail.com>
Subject: and one more
From: Phil <phil@example.com>
To: mytopic@ntfy.sh
Content-Type: text/plain; charset="UTF-8"

you know this is a string.
it's a long string.
it's supposed to be longer than the max message length
which is 4096 bytes,
it used to be 512 bytes, but I increased that for the UnifiedPush support
the 512 bytes was a little short, some people said
but it kinda makes sense when you look at what it looks like one a phone
heck this wasn't even half of it so far.
so i'm gonna fill the rest of this with AAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAa
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
......................................................................
......................................................................
......................................................................
......................................................................
......................................................................
......................................................................
......................................................................
......................................................................
......................................................................
......................................................................
......................................................................
......................................................................
......................................................................
......................................................................
......................................................................
......................................................................
......................................................................
......................................................................
......................................................................
......................................................................
......................................................................
......................................................................
......................................................................
......................................................................
......................................................................
......................................................................
......................................................................
......................................................................
......................................................................
......................................................................
......................................................................
......................................................................
......................................................................
......................................................................
......................................................................
......................................................................
......................................................................
......................................................................
......................................................................
......................................................................
......................................................................
......................................................................
......................................................................
......................................................................
......................................................................
......................................................................
......................................................................
......................................................................
and with BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB
BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB
BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB
that should do it
`
	conf, backend := newTestBackend(t, func(w http.ResponseWriter, r *http.Request) {
		expected := `you know this is a string.
it's a long string.
it's supposed to be longer than the max message length
which is 4096 bytes,
it used to be 512 bytes, but I increased that for the UnifiedPush support
the 512 bytes was a little short, some people said
but it kinda makes sense when you look at what it looks like one a phone
heck this wasn't even half of it so far.
so i'm gonna fill the rest of this with AAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAa
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
......................................................................
......................................................................
......................................................................
......................................................................
......................................................................
......................................................................
......................................................................
......................................................................
......................................................................
......................................................................
......................................................................
......................................................................
......................................................................
......................................................................
......................................................................
......................................................................
......................................................................
......................................................................
......................................................................
......................................................................
......................................................................
......................................................................
......................................................................
......................................................................
......................................................................
......................................................................
......................................................................
......................................................................
......................................................................
......................................................................
......................................................................
......................................................................
......................................................................
......................................................................
......................................................................
......................................................................
......................................................................
......................................................................
......................................................................
......................................................................
......................................................................
......................................................................
......................................................................
......................................................................
......................................................................
......................................................................
......................................................................
......................................................................
and with BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB
BBBBBBBBBBBBBBBBBBBBBBBBB`
		require.Equal(t, 4096, len(expected)) // Sanity check
		require.Equal(t, expected, readAll(t, r.Body))
	})
	conf.SMTPServerAddrPrefix = ""
	session, _ := backend.AnonymousLogin(fakeConnState(t, "1.2.3.4"))
	require.Nil(t, session.Mail("phil@example.com", smtp.MailOptions{}))
	require.Nil(t, session.Rcpt("mytopic@ntfy.sh"))
	require.Nil(t, session.Data(strings.NewReader(email)))
}

func TestSmtpBackend_Unsupported(t *testing.T) {
	email := `Date: Tue, 28 Dec 2021 00:30:10 +0100
Message-ID: <CAAvm79YP0C=Rt1N=KWmSUBB87KK2rRChmdzKqF1vCwMEUiVzLQ@mail.gmail.com>
Subject: and one more
From: Phil <phil@example.com>
To: mytopic@ntfy.sh
Content-Type: text/SOMETHINGELSE

what's up
`
	conf, backend := newTestBackend(t, func(http.ResponseWriter, *http.Request) {
		// Nothing.
	})
	conf.SMTPServerAddrPrefix = ""
	session, _ := backend.Login(fakeConnState(t, "1.2.3.4"), "user", "pass")
	require.Nil(t, session.Mail("phil@example.com", smtp.MailOptions{}))
	require.Nil(t, session.Rcpt("mytopic@ntfy.sh"))
	require.Equal(t, errUnsupportedContentType, session.Data(strings.NewReader(email)))
}

func newTestBackend(t *testing.T, handler func(http.ResponseWriter, *http.Request)) (*Config, *smtpBackend) {
	conf := newTestConfig(t)
	conf.SMTPServerListen = ":25"
	conf.SMTPServerDomain = "ntfy.sh"
	conf.SMTPServerAddrPrefix = "ntfy-"
	backend := newMailBackend(conf, handler)
	return conf, backend
}

func readAll(t *testing.T, rc io.ReadCloser) string {
	b, err := io.ReadAll(rc)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func fakeConnState(t *testing.T, remoteAddr string) *smtp.ConnectionState {
	ip, err := net.ResolveIPAddr("ip", remoteAddr)
	if err != nil {
		t.Fatal(err)
	}
	return &smtp.ConnectionState{
		Hostname:   "myhostname",
		LocalAddr:  ip,
		RemoteAddr: ip,
	}
}
