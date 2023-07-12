
window.im = {}
// 配置
window.limconfig = {};
// 错误回调函数
window.errCallbackFunc;

//iOS  注册事件监听
function setupWebViewJavascriptBridge(callback) {

    if (window.WebViewJavascriptBridge) {
        return callback(WebViewJavascriptBridge);
    }
    if (window.WVJBCallbacks) {
        return window.WVJBCallbacks.push(callback);
    }
    window.WVJBCallbacks = [callback];
    var WVJBIframe = document.createElement('iframe');
    WVJBIframe.style.display = 'none';
    WVJBIframe.src = 'https://__bridge_loaded__';
    document.documentElement.appendChild(WVJBIframe);
    setTimeout(function () {
        document.documentElement.removeChild(WVJBIframe)
    }, 0)
}


//android 注册事件监听
function connectWebViewJavascriptBridge(callback) {

    if (window.WebViewJavascriptBridge) {
        callback(WebViewJavascriptBridge)
    } else {
        document.addEventListener(
            'WebViewJavascriptBridgeReady'
            , function () {
                callback(WebViewJavascriptBridge)
            },
            false
        );
    }
}

/**
通过config接口注入权限验证配置
所有需要使用JS-SDK的页面必须先注入配置信息，否则将无法调用
（同一个url仅需调用一次，对于变化url的SPA的web app可在每次url变化时进行调用）。
**/
im.config = function (cfg) {
    window.limconfig = cfg;
}
im.onError = function (errFunc) {
    window.errCallbackFunc = errFunc;
}

/**
 *  初始化
 * @param {*} callback 
 */
im.onReady = function (callback) {

    setupWebViewJavascriptBridge(function (bridge) {
        window.IMJSBridge = bridge
        if (callback) {
            callback()
        }

    })

    connectWebViewJavascriptBridge(function (bridge) {
        bridge.init(function (message, responseCallback) {
            responseCallback();
        });
        window.IMJSBridge = bridge
        if (callback) {
            callback()
        }
    })
}


/**
  调用方法
  method: 方法名
  options 参数
  successCallback: 成功回调
  errorCallback：错误回调
  completeCallback: 完成回调
**/
im.call = function (method, params) {
    if (window.IMJSBridge) {
        window.IMJSBridge.callHandler(method, params ? params.options : undefined, function (response) {
            let result = JSON.parse(response)
            if (result.err_code == undefined || result.err_code == 200) { // 正确请求
                if (params.success) {
                    params.success(result)
                }
            } else {
                // 错误
                if (params.error) {
                    params.error(result)
                }
            }
            // 完成
            if (params.complete) {
                params.complete(result)
            }
        })
    }
}

// ---------- 常用函数 ----------

// 退出webview
im.quit = function () {
    im.call("quit");
}

// 获取频道信息
im.getChannel = function () {
    return new Promise(function (resolve, reject) {
        im.call("getChannel", {
            success: function (channel) {
                resolve(channel);
            },
            error: function (result) {
                reject(result);
            }
        });
    });
}

// 退出webview
im.quit = function () {
    im.call("quit");
}


// 显示最近会话
im.showConversation = function (channelID, channelType) {
    im.call("showConversation", {
        options: {
            'forward': 'replace',
            'channel_id': channelID,
            'channel_type': channelType
        }
    })
}