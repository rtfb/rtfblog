function inputToUri(fieldName) {
    var rawValue = document.getElementsByName(fieldName)[0].value;
    return fieldName + "=" + encodeURIComponent(rawValue);
}

function validateForm() {
    var name = document.getElementById('name').value;
    if (name === "") {
        alert("Name field is mandatory.");
        return false;
    }
    var email = document.getElementById('email').value;
    if (email === "") {
        alert("Email field is mandatory.");
        return false;
    }
    return true;
}

function submitComment() {
    if (!validateForm()) {
        return;
    }

    var xhr;
    try {
        xhr = new ActiveXObject('Msxml2.XMLHTTP');
    } catch (e) {
        try {
            xhr = new ActiveXObject('Microsoft.XMLHTTP');
        } catch (e2) {
            try {
                xhr = new XMLHttpRequest();
            } catch (e3) {
                xhr = false;
            }
        }
    }

    xhr.onreadystatechange = function() {
        if (xhr.readyState == 4) {
            if (xhr.status == 200) {
                var response = JSON.parse(xhr.responseText);
                if (response["status"] === "rejected") {
                    document.getElementById('captcha-input').value = '';
                } else if (response["status"] === "showcaptcha") {
                    document.getElementById('captcha-alert-box').style.visibility = 'visible';
                    document.getElementById('captcha-id').value = response["captcha-id"];
                } else {
                    window.location.href = response["redir"];
                    window.location.reload(true);
                }
            } else {
                alert("Error submitting comment. Status = " + xhr.status);
            }
        }
    };

    try {
        var params = inputToUri('name');
        params += "&" + inputToUri('captcha-id');
        params += "&" + inputToUri('captcha');
        params += "&" + inputToUri('email');
        params += "&" + inputToUri('website');
        params += "&" + inputToUri('text');
        xhr.open("GET", "comment_submit?" + params, true);
        xhr.send(null);
    } catch (err) {
        alert("exc: " + err);
    }
}

global.window.submitComment = submitComment;
