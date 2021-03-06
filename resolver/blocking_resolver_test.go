package resolver

import (
	"blocky/config"
	"blocky/helpertest"
	"blocky/util"
	"net"
	"testing"

	"github.com/miekg/dns"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func Test_Resolve_ClientName_IpZero(t *testing.T) {
	file := helpertest.TempFile("blocked1.com")
	defer file.Close()

	sut := NewBlockingResolver(config.BlockingConfig{
		BlackLists: map[string][]string{"gr1": {file.Name()}},
		ClientGroupsBlock: map[string][]string{
			"client1": {"gr1"},
		},
	})
	req := util.NewMsgWithQuestion("blocked1.com.", dns.TypeA)

	// A
	resp, err := sut.Resolve(&Request{
		Req:         req,
		ClientNames: []string{"client1"},
		ClientIP:    net.ParseIP("192.168.178.55"),
		Log:         logrus.NewEntry(logrus.New()),
	})

	assert.NoError(t, err)
	assert.Equal(t, dns.RcodeSuccess, resp.Res.Rcode)
	assert.Equal(t, "blocked1.com.	21600	IN	A	0.0.0.0", resp.Res.Answer[0].String())

	// AAAA
	req = util.NewMsgWithQuestion("blocked1.com.", dns.TypeAAAA)
	resp, err = sut.Resolve(&Request{
		Req:         req,
		ClientNames: []string{"client1"},
		ClientIP:    net.ParseIP("192.168.178.55"),
		Log:         logrus.NewEntry(logrus.New()),
	})
	assert.NoError(t, err)
	assert.Equal(t, dns.RcodeSuccess, resp.Res.Rcode)
	assert.Equal(t, "blocked1.com.	21600	IN	AAAA	::", resp.Res.Answer[0].String())
}

func Test_Resolve_ClientIp_A_IpZero(t *testing.T) {
	file := helpertest.TempFile("blocked1.com")
	defer file.Close()

	sut := NewBlockingResolver(config.BlockingConfig{
		BlackLists: map[string][]string{"gr1": {file.Name()}},
		ClientGroupsBlock: map[string][]string{
			"192.168.178.55": {"gr1"},
		},
	})

	req := util.NewMsgWithQuestion("blocked1.com.", dns.TypeA)

	resp, err := sut.Resolve(&Request{
		Req:         req,
		ClientNames: []string{"unknown"},
		ClientIP:    net.ParseIP("192.168.178.55"),
		Log:         logrus.NewEntry(logrus.New()),
	})
	assert.NoError(t, err)
	assert.Equal(t, dns.RcodeSuccess, resp.Res.Rcode)
	assert.Equal(t, "blocked1.com.	21600	IN	A	0.0.0.0", resp.Res.Answer[0].String())
}

func Test_Resolve_ClientWith2Names_A_IpZero(t *testing.T) {
	file1 := helpertest.TempFile("blocked1.com")
	defer file1.Close()

	file2 := helpertest.TempFile("blocked2.com")
	defer file2.Close()

	sut := NewBlockingResolver(config.BlockingConfig{
		BlackLists: map[string][]string{
			"gr1": {file1.Name()},
			"gr2": {file2.Name()},
		},
		ClientGroupsBlock: map[string][]string{
			"client1": {"gr1"},
			"altName": {"gr2"},
		},
	})

	// request in gr1
	req := util.NewMsgWithQuestion("blocked1.com.", dns.TypeA)
	resp, err := sut.Resolve(&Request{
		Req:         req,
		ClientNames: []string{"client1", "altName"},
		ClientIP:    net.ParseIP("192.168.178.55"),
		Log:         logrus.NewEntry(logrus.New()),
	})
	assert.NoError(t, err)
	assert.Equal(t, dns.RcodeSuccess, resp.Res.Rcode)
	assert.Equal(t, "blocked1.com.	21600	IN	A	0.0.0.0", resp.Res.Answer[0].String())

	// request in gr2
	req = util.NewMsgWithQuestion("blocked2.com.", dns.TypeA)
	resp, err = sut.Resolve(&Request{
		Req:         req,
		ClientNames: []string{"client1", "altName"},
		ClientIP:    net.ParseIP("192.168.178.55"),
		Log:         logrus.NewEntry(logrus.New()),
	})
	assert.NoError(t, err)
	assert.Equal(t, dns.RcodeSuccess, resp.Res.Rcode)
	assert.Equal(t, "blocked2.com.	21600	IN	A	0.0.0.0", resp.Res.Answer[0].String())
}

func Test_Resolve_Default_A_IpZero(t *testing.T) {
	file := helpertest.TempFile("blocked1.com")
	defer file.Close()

	sut := NewBlockingResolver(config.BlockingConfig{
		BlackLists: map[string][]string{"gr1": {file.Name()}},
		ClientGroupsBlock: map[string][]string{
			"default": {"gr1"},
		},
	})

	req := util.NewMsgWithQuestion("blocked1.com.", dns.TypeA)
	resp, err := sut.Resolve(&Request{
		Req:         req,
		ClientNames: []string{"unknown"},
		ClientIP:    net.ParseIP("192.168.178.1"),
		Log:         logrus.NewEntry(logrus.New()),
	})
	assert.NoError(t, err)
	assert.Equal(t, dns.RcodeSuccess, resp.Res.Rcode)
	assert.Equal(t, "blocked1.com.	21600	IN	A	0.0.0.0", resp.Res.Answer[0].String())
}

func Test_Resolve_Default_Block_With_Whitelist(t *testing.T) {
	file := helpertest.TempFile("blocked1.com")
	defer file.Close()

	sut := NewBlockingResolver(config.BlockingConfig{
		BlackLists: map[string][]string{"gr1": {file.Name()}},
		WhiteLists: map[string][]string{"gr1": {file.Name()}},
		ClientGroupsBlock: map[string][]string{
			"default": {"gr1"},
		},
	})

	m := &resolverMock{}
	m.On("Resolve", mock.Anything).Return(new(Response), nil)
	sut.Next(m)

	req := util.NewMsgWithQuestion("blocked1.com.", dns.TypeA)
	_, err := sut.Resolve(&Request{
		Req:         req,
		ClientNames: []string{"unknown"},
		ClientIP:    net.ParseIP("192.168.178.1"),
		Log:         logrus.NewEntry(logrus.New()),
	})
	assert.NoError(t, err)
	m.AssertExpectations(t)
}

func Test_Resolve_Whitelist_Only(t *testing.T) {
	file := helpertest.TempFile("whitelisted.com")
	defer file.Close()

	sut := NewBlockingResolver(config.BlockingConfig{
		WhiteLists: map[string][]string{"gr1": {file.Name()}},
		ClientGroupsBlock: map[string][]string{
			"default": {"gr1"},
		},
	})

	m := &resolverMock{}
	m.On("Resolve", mock.Anything).Return(new(Response), nil)
	sut.Next(m)

	req := util.NewMsgWithQuestion("whitelisted.com.", dns.TypeA)
	_, err := sut.Resolve(&Request{
		Req:         req,
		ClientNames: []string{"unknown"},
		ClientIP:    net.ParseIP("192.168.178.1"),
		Log:         logrus.NewEntry(logrus.New()),
	})
	assert.NoError(t, err)
	m.AssertExpectations(t)

	req = new(dns.Msg)
	req.SetQuestion("google.com.", dns.TypeA)

	resp, err := sut.Resolve(&Request{
		Req:         req,
		ClientNames: []string{"unknown"},
		ClientIP:    net.ParseIP("192.168.178.1"),
		Log:         logrus.NewEntry(logrus.New()),
	})

	assert.NoError(t, err)
	assert.Equal(t, dns.RcodeSuccess, resp.Res.Rcode)
	assert.Equal(t, "google.com.	21600	IN	A	0.0.0.0", resp.Res.Answer[0].String())
	assert.Equal(t, 1, len(m.Calls))
}

func Test_determineWhitelistOnlyGroups(t *testing.T) {
	assert.Equal(t, []string{"w1"}, determineWhitelistOnlyGroups(&config.BlockingConfig{
		BlackLists: map[string][]string{},
		WhiteLists: map[string][]string{"w1": {"l1"}},
	}))

	assert.Equal(t, []string{"b1", "default"}, determineWhitelistOnlyGroups(&config.BlockingConfig{
		BlackLists: map[string][]string{
			"w1": {"y"},
		},
		WhiteLists: map[string][]string{
			"w1":      {"l1"},
			"default": {"s1"},
			"b1":      {"x"}},
	}))
}

func Test_Resolve_Default_A_NxRecord(t *testing.T) {
	file := helpertest.TempFile("blocked1.com")
	defer file.Close()

	sut := NewBlockingResolver(config.BlockingConfig{
		BlackLists: map[string][]string{"gr1": {file.Name()}},
		ClientGroupsBlock: map[string][]string{
			"default": {"gr1"},
		},
		BlockType: "NxDomain",
	})

	req := util.NewMsgWithQuestion("blocked1.com.", dns.TypeA)
	resp, err := sut.Resolve(&Request{
		Req:         req,
		ClientNames: []string{"unknown"},
		ClientIP:    net.ParseIP("192.168.178.1"),
		Log:         logrus.NewEntry(logrus.New()),
	})
	assert.NoError(t, err)
	assert.Equal(t, dns.RcodeNameError, resp.Res.Rcode)
}

func Test_Resolve_NoBlock(t *testing.T) {
	file := helpertest.TempFile("blocked1.com")
	defer file.Close()

	sut := NewBlockingResolver(config.BlockingConfig{
		BlackLists: map[string][]string{"gr1": {file.Name()}},
		ClientGroupsBlock: map[string][]string{
			"client1": {"gr1"},
		},
	})

	m := &resolverMock{}
	m.On("Resolve", mock.Anything).Return(new(Response), nil)
	sut.Next(m)

	req := util.NewMsgWithQuestion("example.com.", dns.TypeA)
	_, err := sut.Resolve(&Request{
		Req:         req,
		ClientNames: []string{"unknown"},
		ClientIP:    net.ParseIP("192.168.178.1"),
		Log:         logrus.NewEntry(logrus.New()),
	})
	assert.NoError(t, err)
	m.AssertExpectations(t)
}

func Test_Configuration_BlockingResolver(t *testing.T) {
	file := helpertest.TempFile("blocked1.com")
	defer file.Close()

	sut := NewBlockingResolver(config.BlockingConfig{
		BlackLists: map[string][]string{"gr1": {file.Name()}},
		WhiteLists: map[string][]string{"gr1": {file.Name()}},
		ClientGroupsBlock: map[string][]string{
			"default": {"gr1"},
		},
	})

	c := sut.Configuration()
	assert.True(t, len(c) > 1)
}

func Test_Resolve_WrongBlockType(t *testing.T) {
	defer func() { logrus.StandardLogger().ExitFunc = nil }()

	var fatal bool

	logrus.StandardLogger().ExitFunc = func(int) { fatal = true }

	_ = NewBlockingResolver(config.BlockingConfig{
		BlockType: "wrong",
	})

	assert.True(t, fatal)
}

func Test_Resolve_NoLists(t *testing.T) {
	sut := NewBlockingResolver(config.BlockingConfig{})
	m := &resolverMock{}
	m.On("Resolve", mock.Anything).Return(new(Response), nil)
	sut.Next(m)

	req := util.NewMsgWithQuestion("example.com.", dns.TypeA)
	_, err := sut.Resolve(&Request{
		Req:         req,
		ClientNames: []string{"unknown"},
		ClientIP:    net.ParseIP("192.168.178.1"),
		Log:         logrus.NewEntry(logrus.New()),
	})
	assert.NoError(t, err)
	m.AssertExpectations(t)

	c := sut.Configuration()

	assert.Equal(t, []string{"deactivated"}, c)
}
