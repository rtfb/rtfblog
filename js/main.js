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

function validateCommentForm() {
    var name = elt('name').value;
    if (name === "") {
        myAlert("Name field is mandatory.");
        return false;
    }
    var email = elt('email').value;
    if (email === "") {
        myAlert("Email field is mandatory.");
        return false;
    }
    return true;
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

global.window.uploadProgress = uploadProgress;
global.window.forwardClickToFileid = forwardClickToFileid;
global.window.submitComment = submitComment;
