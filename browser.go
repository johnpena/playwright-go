package playwright

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type browserImpl struct {
	channelOwner
	isConnected                  bool
	isClosedOrClosing            bool
	shouldCloseConnectionOnClose bool
	contexts                     []BrowserContext
	browserType                  BrowserType
	chromiumTracingPath          *string
}

func (b *browserImpl) BrowserType() BrowserType {
	return b.browserType
}

func (b *browserImpl) IsConnected() bool {
	b.RLock()
	defer b.RUnlock()
	return b.isConnected
}

func (b *browserImpl) NewContext(options ...BrowserNewContextOptions) (BrowserContext, error) {
	overrides := map[string]interface{}{}
	option := BrowserNewContextOptions{}
	if len(options) == 1 {
		option = options[0]
	}
	if option.ExtraHttpHeaders != nil {
		overrides["extraHTTPHeaders"] = serializeMapToNameAndValue(options[0].ExtraHttpHeaders)
		options[0].ExtraHttpHeaders = nil
	}
	if option.StorageStatePath != nil {
		var storageState *OptionalStorageState
		storageString, err := os.ReadFile(*options[0].StorageStatePath)
		if err != nil {
			return nil, fmt.Errorf("could not read storage state file: %w", err)
		}
		err = json.Unmarshal(storageString, &storageState)
		if err != nil {
			return nil, fmt.Errorf("could not parse storage state file: %w", err)
		}
		options[0].StorageState = storageState
		options[0].StorageStatePath = nil
	}
	if option.NoViewport != nil && *options[0].NoViewport {
		overrides["noDefaultViewport"] = true
		options[0].NoViewport = nil
	}
	if option.RecordHarPath != nil {
		overrides["recordHar"] = prepareRecordHarOptions(recordHarInputOptions{
			Path:        *options[0].RecordHarPath,
			URL:         options[0].RecordHarURLFilter,
			Mode:        options[0].RecordHarMode,
			Content:     options[0].RecordHarContent,
			OmitContent: options[0].RecordHarOmitContent,
		})
		options[0].RecordHarPath = nil
		options[0].RecordHarURLFilter = nil
		options[0].RecordHarMode = nil
		options[0].RecordHarContent = nil
		options[0].RecordHarOmitContent = nil
	}
	channel, err := b.channel.Send("newContext", overrides, options)
	if err != nil {
		return nil, fmt.Errorf("could not send message: %w", err)
	}
	context := fromChannel(channel).(*browserContextImpl)
	context.browser = b
	b.browserType.(*browserTypeImpl).didCreateContext(context, &option, nil)
	return context, nil
}

func (b *browserImpl) NewPage(options ...BrowserNewPageOptions) (Page, error) {
	opts := make([]BrowserNewContextOptions, 0)
	if len(options) == 1 {
		opts = append(opts, BrowserNewContextOptions(options[0]))
	}
	context, err := b.NewContext(opts...)
	if err != nil {
		return nil, err
	}
	page, err := context.NewPage()
	if err != nil {
		return nil, err
	}
	page.(*pageImpl).ownedContext = context
	context.(*browserContextImpl).ownedPage = page
	return page, nil
}

func (b *browserImpl) NewBrowserCDPSession() (CDPSession, error) {
	channel, err := b.channel.Send("newBrowserCDPSession")
	if err != nil {
		return nil, fmt.Errorf("could not send message: %w", err)
	}

	cdpSession := fromChannel(channel).(*cdpSessionImpl)

	return cdpSession, nil
}

func (b *browserImpl) Contexts() []BrowserContext {
	b.Lock()
	defer b.Unlock()
	return b.contexts
}

func (b *browserImpl) Close() error {
	if b.isClosedOrClosing {
		return nil
	}
	b.Lock()
	b.isClosedOrClosing = true
	b.Unlock()
	_, err := b.channel.Send("close")
	if err != nil && !isSafeCloseError(err) {
		return fmt.Errorf("close browser failed: %w", err)
	}
	if b.shouldCloseConnectionOnClose {
		return b.connection.Stop()
	}
	return nil
}

func (b *browserImpl) Version() string {
	return b.initializer["version"].(string)
}

func (b *browserImpl) StartTracing(options ...BrowserStartTracingOptions) error {
	overrides := map[string]interface{}{}
	option := BrowserStartTracingOptions{}
	if len(options) == 1 {
		option = options[0]
	}
	if option.Page != nil {
		overrides["page"] = option.Page.(*pageImpl).channel
		option.Page = nil
	}
	if option.Path != nil {
		b.chromiumTracingPath = option.Path
		option.Path = nil
	}
	_, err := b.channel.Send("startTracing", option, overrides)
	return err
}

func (b *browserImpl) StopTracing() ([]byte, error) {
	channel, err := b.channel.Send("stopTracing")
	if err != nil {
		return nil, err
	}
	artifact := fromChannel(channel).(*artifactImpl)
	binary, err := artifact.ReadIntoBuffer()
	if err != nil {
		return nil, err
	}
	err = artifact.Delete()
	if err != nil {
		return binary, err
	}
	if b.chromiumTracingPath != nil {
		err := os.MkdirAll(filepath.Dir(*b.chromiumTracingPath), 0777)
		if err != nil {
			return binary, err
		}
		err = os.WriteFile(*b.chromiumTracingPath, binary, 0644)
		if err != nil {
			return binary, err
		}
	}
	return binary, nil
}

func (b *browserImpl) onClose() {
	b.Lock()
	b.isClosedOrClosing = true
	if b.isConnected {
		b.isConnected = false
		b.Unlock()
		b.Emit("disconnected", b)
		return
	}
	b.Unlock()
}

func (b *browserImpl) OnDisconnected(fn func(Browser)) {
	b.On("disconnected", fn)
}

func newBrowser(parent *channelOwner, objectType string, guid string, initializer map[string]interface{}) *browserImpl {
	b := &browserImpl{
		isConnected: true,
		contexts:    make([]BrowserContext, 0),
	}
	b.createChannelOwner(b, parent, objectType, guid, initializer)
	// convert parent to *browserTypeImpl
	b.browserType = newBrowserType(parent.parent, parent.objectType, parent.guid, parent.initializer)
	b.channel.On("close", b.onClose)
	return b
}
