new Vue({
    el: '#app',

    data: {
        ws: null, // Our websocket
        newMsg: '', // Holds new messages to be sent to the server
        chatContent: '', // A running list of chat messages displayed on the screen
        username: null, // Our username
        joined: false // True if email and username have been filled in
    },

    created: function() {
        var self = this;
        this.ws = new WebSocket('ws://' + window.location.host + '/ws');
        this.ws.addEventListener('message', function(e) {
            console.log("message:", e);
            var msg = JSON.parse(e.data);
            msg = msg.message;
            if (msg.filelink == "") {
                self.chatContent += '<div class="chip">' + msg.username + '</div>'
                    + emojione.toImage(msg.message) + '<br/>' + '<div class="time" style="color:gray">' + self.getCurrentTime() + '</div>';
            } else {
                self.chatContent += '<div class="chip">'
                        + msg.username + '</div>' 
                    + emojione.toImage(msg.message) + '<br/>' + '&nbsp;&nbsp;&nbsp;&nbsp;<a href=' + msg.filelink + '>' 
                    + msg.filelink + '</a>' + '<div class="time" style="color:gray">' + self.getCurrentTime() + '</div>';
            }

            var element = document.getElementById('chat-messages');
            element.scrollTop = element.scrollHeight; // Auto scroll to the bottom
        });
    },

    methods: {
        send: function () {
            var curTime = this.getCurrentTime();
            if (this.newMsg != '') {
                this.ws.send(
                    JSON.stringify({
                        username: this.username,
                        message: $('<p>').html(this.newMsg).text(), // Strip out html
                        time: curTime
                    }
                ));
                this.newMsg = ''; // Reset newMsg
            }
        },

        join: function () {
            if (!this.username) {
                Materialize.toast('You must choose a username', 2000);
                return
            }
            this.username = $('<p>').html(this.username).text();
            this.joined = true;
        },

        gravatarURL: function(email) {
            //return 'http://www.gravatar.com/avatar/' + CryptoJS.MD5('');
        },

        getCurrentTime: function() {
            var date = new Date();
            var year = date.getFullYear();
            var month = date.getMonth() + 1;
            var day = date.getDate();
            var hour = date.getHours();
            var minute = date.getMinutes();
            var second = date.getSeconds()
            return year + '-' + month + '-' + day + ' ' + hour + ':' + minute + ':' + second;
        }
    }
});