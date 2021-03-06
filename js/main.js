function inputToUri(fieldName) {
    var rawValue = document.getElementsByName(fieldName)[0].value;
    return fieldName + "=" + encodeURIComponent(rawValue);
}

function elt(id) {
    return document.getElementById(id);
}

function myAlert(msg) {
    alert(msg);
}

function validateField(id, msg) {
    var val = elt(id).value;
    if (val === "") {
        myAlert(msg);
        return false;
    }
    return true;
}

function validateCommentForm() {
    return validateField('name', "Name field is mandatory.")
        && validateField('email', "Email field is mandatory.");
}

function validatePostForm() {
    return validateField('post_title', "Title field is mandatory.")
        && validateField('post_url', "URL field is mandatory.")
        && validateField('wmd-input', "Post content is mandatory.");
}

function validateAuthorForm() {
    return validateField("author_username", "User name is mandatory.")
        && validateField("author_displayname", "Display name is mandatory.")
        // optional: && validateField("author_email", "Email is mandatory.")
        // optional: && validateField("author_www", "Web site is mandatory.")
        && validateField("author_password", "Password is mandatory.")
        && validateField("author_confirm_password", "Password is mandatory.");
}

function mkXHR() {
    try {
        return new ActiveXObject('Msxml2.XMLHTTP');
    } catch (e) {
        try {
            return new ActiveXObject('Microsoft.XMLHTTP');
        } catch (e2) {
            try {
                return new XMLHttpRequest();
            } catch (e3) {
                return false;
            }
        }
    }
};

function scrollIntoView(id) {
    var e = elt(id);
    if (!!e && e.scrollIntoView) {
        e.scrollIntoView();
    }
}

function submitComment() {
    if (!validateCommentForm()) {
        return;
    }

    var xhr = mkXHR();
    xhr.onreadystatechange = function() {
        if (xhr.readyState == 4) {
            if (xhr.status == 200) {
                var response = JSON.parse(xhr.responseText);
                if (response["status"] === "rejected") {
                    elt('captcha-input').value = '';
                } else if (response["status"] === "showcaptcha") {
                    var task = response["captcha-task"];
                    elt('captcha-task-text').textContent = task;
                    elt('captcha-alert-box').style.visibility = 'visible';
                    elt('captcha-id').value = response["captcha-id"];
                    scrollIntoView('captcha-alert-box');
                    elt('captcha-input').focus();
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

var uploadNo = 0;

function forwardClickToFileid() {
    var fileid = elt('fileid');
    fileid.click();
}

function mkUploadHtml(filename, uploadNo) {
    var p = document.createElement('p');
    p.setAttribute('id', 'progress_' + (uploadNo + 1));
    // TODO: .textContent doesn't work on IE, need to use .innerText
    p.textContent = filename;
    return p;
}

function uploadProgress() {
    var filename = this.value.split('\\').pop();
    if (!filename)
        return;
    var formData = new FormData(elt('edit-post-form'));
    console.log(JSON.stringify(formData));
    var uploadSection = elt('upload-progress-section');
    uploadSection.appendChild(mkUploadHtml(filename, uploadNo));
    var xhr = mkXHR();
    xhr.onreadystatechange = function() {
        if (xhr.readyState == 4) {
            if (xhr.status == 200) {
                var postTextarea = elt('wmd-input');
                postTextarea.innerHTML += "\n" + xhr.responseText;
            } else {
                alert("Error uploading: " + xhr.status);
            }
        }
    };
    xhr.upload.onprogress = function(evt) {
        console.log("progrFunc");
        if (evt.lengthComputable) {
            var p = elt('progress_' + uploadNo);
            var pc = parseInt(100 - (evt.loaded / evt.total * 100));
            p.style.backgroundPosition = pc + "% 0";
        }
    }
    try {
        xhr.open("POST", "upload_images", true);
        xhr.send(formData);
    } catch (err) {
        alert("exc: " + err);
    }
    uploadNo += 1;
}

function removeElt(id) {
    var elem = document.getElementById(id);
    if (elem) {
        elem.parentNode.removeChild(elem);
    }
}

window.uploadProgress = uploadProgress;
window.forwardClickToFileid = forwardClickToFileid;
window.submitComment = submitComment;
window.validatePostForm = validatePostForm;
window.validateAuthorForm = validateAuthorForm;
window.removeElt = removeElt;
