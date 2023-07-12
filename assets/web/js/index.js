
// TODO: 此页面的 invoke 相关的方法应该作废了 应该使用IMJSBridge这个JS

/**
 *  显示会话界面
 * @param {*} param0 
 */
function showConversation({channelID,channelType}) {
    invoke('showConversation',{
        'forward': 'replace',
        'channel_id': channelID,
        'channel_type': channelType
    })
}

function pop() {
    invoke('pop')
}

// 关闭
function closeWebView() {
    invoke('pop')
}

function getQueryString(name)
{
     var reg = new RegExp("(^|&)"+ name +"=([^&]*)(&|$)");
     var r = window.location.search.substr(1).match(reg);
     if(r!=null)return  unescape(r[2]); return null;
}


function invoke(method,param,callback) {
    if(isIOS()) {
        invokeIOS(method,param,callback);
    }
}

function invokeIOS(method,param,callback) {
     window.WKJSBridge.callNative("LIMCommonPlugin", method, param, function success(res) {
         if(callback) {
            callback(true,res)
         }
     }, function fail(res) {
        if(callback) {
            callback(false,res)
         }
     });
}

function isIOS() {
    return true
}

function isAndroid() {
    return false
}

$.extend({
    postJSON: function (url, body) {
        return $.ajax({
            type: 'POST',
            url: url,
            data: JSON.stringify(body),
            contentType: "application/json",
            dataType: 'json'
        });
    }
});

function uuid() {
	var s = [];
	var hexDigits = "0123456789abcdef";
	for (var i = 0; i < 36; i++) {
		s[i] = hexDigits.substr(Math.floor(Math.random() * 0x10), 1);
	}
	s[14] = "4"; // bits 12-15 of the time_hi_and_version field to 0010
	s[19] = hexDigits.substr((s[19] & 0x3) | 0x8, 1); // bits 6-7 of the clock_seq_hi_and_reserved to 01
	s[8] = s[13] = s[18] = s[23] = "-";

	var uuid = s.join("");
	return uuid;
}