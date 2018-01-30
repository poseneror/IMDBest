
var roomID = $("#roomID").text();
var playerID = $("#playerID").text();
var ws = new WebSocket("ws://localhost:8080/ws/" + roomID + playerID);
var scoreFrame = $("#ScoreFrame");
var questionFrame = $("#QuestionFrame");
var resumeFrame = $("#ResumeFrame");
var clockFrame = $("#Clock");
var scoreList = $("#ScoreList");
var playersList = $("#playersList");
var setupFrame = $("#SetupFrame");
var movieFrame = $("#MovieFrame");
var gameOverFrame = $("#GameOverFrame");
var waitFrame = $("#WaitFrame");
var movieTitle = $("#MovieTitle");
var moviePoster = $("#MoviePoster");
var expectedAnswer = $("#ExpectedAnswer");
var currentAnswer = $("#CurrentAnswer");
var setupSelectionFrame = $("#SetupSelectionFrame");
var timerText = $("#Timer");
var seconds = $('.seconds');
var setupSelection = $("#SetupSelection");
var frames =$(".frame");
var gameFrame = $("#GameFrame");

var current;
var question;
var mode;
var players;
var setup;
var timer;
var history;
var next;

// game modes
const MODE_SETUP = 1;
const MODE_INGAME = 2;
const MODE_BREAK = 3;
const MODE_GAMEOVER = 4;

// question types
const TYPE_RATING = 1;
const TYPE_CAST = 2;
const TYPE_CHARACTER = 3;
const TYPE_BUDJET = 4;

// message types
const TYPE_UPDATE_ROOM = 1;
const TYPE_START_GAME = 2;
const TYPE_PLAYER_JOIN = 3;
const TYPE_PLAYER_LEAVE = 4;
const TYPE_PLAYER_ANSWER = 5;
const TYPE_END_GAME = 6;
const TYPE_CONTINUE_GAME = 7;
const TYPE_BREAK_GAME = 8;
const TYPE_GAME_MODE_CHANGE = 9;
const TYPE_SETUP_CATEGORY = 10;
const TYPE_QUESTION_FETCHED = 11;
const TYPE_PROMOTED = 12;

// Categories
const CATEGORY_POPULAR = 1;
const CATEGORY_COMPANY = 2;
const CATEGORY_GENERE = 3;

var genres = [
    {
        "id": 28,
        "name": "Action"
    },
    {
        "id": 12,
        "name": "Adventure"
    },
    {
        "id": 16,
        "name": "Animation"
    },
    {
        "id": 35,
        "name": "Comedy"
    },
    {
        "id": 80,
        "name": "Crime"
    },
    {
        "id": 99,
        "name": "Documentary"
    },
    {
        "id": 18,
        "name": "Drama"
    },
    {
        "id": 10751,
        "name": "Family"
    },
    {
        "id": 14,
        "name": "Fantasy"
    },
    {
        "id": 36,
        "name": "History"
    },
    {
        "id": 27,
        "name": "Horror"
    },
    {
        "id": 10402,
        "name": "Music"
    },
    {
        "id": 9648,
        "name": "Mystery"
    },
    {
        "id": 10749,
        "name": "Romance"
    },
    {
        "id": 878,
        "name": "Science Fiction"
    },
    {
        "id": 10770,
        "name": "TV Movie"
    },
    {
        "id": 53,
        "name": "Thriller"
    },
    {
        "id": 10752,
        "name": "War"
    },
    {
        "id": 37,
        "name": "Western"
    }
];

var timeleft;
function showFrame(e) {
    e.removeClass("hidden");
}
function hideFrame(e) {
    e.addClass("hidden");
}

// Players		[]*Player 		`json:"players"`
// Current		*Player			`json:"current"`
// Mode 		int				`json:"mode"`
// Question	Question		`json:"question"`
// Setup	 	*Setup			`json:"setup"`
// Type 		int 			`json:"type"`
ws.onmessage = function (event) {
    var upd = JSON.parse(event.data);
    // alert(upd.type);
    if(upd.current != null){
        current = upd.current;
        if(current.admin){
            switch(mode){
                case MODE_SETUP:
                    showFrame(setupFrame);
                    showFrame(setupSelectionFrame);
                    break;

                case MODE_INGAME:
                    break;

                case MODE_BREAK:
                    showFrame(resumeFrame);
                    break;
                case MODE_GAMEOVER:
                    showFrame(setupFrame);
                    break;
            }

        }
    }
    if(upd.players != null){
        updatePlayerList(upd.players);
    }
    if(upd.mode != 0){
        mode = upd.mode;
        hideFrame(frames);
        switch (mode){
            case MODE_SETUP:
                if(current.admin) {
                    showFrame(setupFrame);
                    hideLoading(setupFrame);
                }
                break;
            case MODE_INGAME:
                showFrame(gameFrame);
                showLoading(gameFrame, "Loading, please wait...");
                questionFrame.empty();
                currentAnswer.empty();
                break;
            case MODE_BREAK:
                showFrame(gameFrame);
                showLoading(gameFrame, "Loading, please wait...");
                expectedAnswer.empty();
                scoreList.empty();
                clearInterval(timer);
                if(current.admin) {
                    showFrame(resumeFrame);
                }
                break;
            case MODE_GAMEOVER:
                showFrame(gameOverFrame);
                if(current.admin){
                    showFrame(setupFrame);
                }
                showLoading(gameOverFrame, "Loading game results..");
                break;
        }
    }
    if(upd.question != null){
        question = upd.question;
        switch (mode){
            case MODE_INGAME:
                hideLoading(gameFrame);
                switch (question.style) {
                    case TYPE_RATING:
                        if(!current.guessed){
                            questionFrame.append("<p class='lead'>Rate it:</p>" +
                                "<input id='InputGuess' type='number' class='rating' data-min=0 data-max=10 data-step=0.1" +
                                " data-size='sm' data-animate='false' data-show-clear='false' data-show-caption='false'/>" +
                                "<span id='InputGuessLabel'></span>");
                            var inputGuessLabel = $("#InputGuessLabel");
                            var inputGuess = $('#InputGuess')
                                .rating().on('rating.hover', function(event, value, caption, target) {
                                    inputGuessLabel.text(value)})
                                .on('rating.hoverleave', function(event, value, caption, target) {
                                    inputGuessLabel.text("")})
                                .on('rating.change', function(event, value, caption) {
                                    submitGuess(value)});
                            showFrame(questionFrame);
                            hideFrame(waitFrame);
                        } else {
                            currentAnswer.append("<h4>" + question.answers[current.id].value + "</h4>");
                            currentAnswer.append("<input type='number' class='rating' data-min=0 " +
                                "data-max=10 data-step=0.1 data-size='xs' data-show-clear='false' " +
                                "data-show-caption='false' data-display-only='true' value='"
                                + question.answers[current.id].value + "'/>");
                            $('.rating').rating('refresh', {});
                            showFrame(waitFrame);
                            hideFrame(questionFrame);
                            clearInterval(timer);
                        }
                        break;
                    case TYPE_CAST:
                        questionFrame.append("<h3>Who acted in this movie?</h3>");
                        if(!current.guessed){
                            for (var j = 0; j < question.options.length; j++) {
                                questionFrame.append("<label class='btn cast-selection' onclick='submitGuess("+j+")'>" +
                                    "<img src='https://image.tmdb.org/t/p/w75/" + question.options[j].profile_path + "'/>" +
                                    "<div class='title'>" + question.options[j].Name + "</div></label>");
                            }
                            hideFrame(questionFrame);
                            showFrame(questionFrame);
                        } else {
                            currentAnswer.append("<h4>" + question.options[question.answers[current.id].value].Name + "</h4>" +
                                "<img src='https://image.tmdb.org/t/p/w75/" +
                                question.options[question.answers[current.id].value].profile_path + "'/>");
                            clearInterval(timer);
                            hideFrame(questionFrame);
                            showFrame(waitFrame);
                        }
                        break;
                }
                timeleft = question.timeleft;
                setClock(timeleft);
                showFrame(clockFrame);
                // update movie details
                movieTitle.text(question.movie.Title);
                moviePoster.attr("src", "https://image.tmdb.org/t/p/w185/" + question.movie.poster_path);
                showFrame(movieFrame);
                break;
            case MODE_BREAK:
                hideLoading(gameFrame);
                movieTitle.text(question.movie.Title);
                moviePoster.attr("src", "https://image.tmdb.org/t/p/w185/" + question.movie.poster_path);
                switch(question.style) {
                    case TYPE_RATING:
                        if(question.answers[current.id].score == 20) {
                            popMessage("RIGHT ON!@&$");
                            $('#CorrectAudio').trigger("pause").prop("currentTime", 0).trigger("play");
                        } else if (question.answers[current.id].score > 8) {
                            popMessage("Not bad :)");
                            $('#CorrectAudio').trigger("pause").prop("currentTime", 0).trigger("play");
                        }
                        expectedAnswer.append("<h4>Actual Movie Rating: " + question.expected + "</h4>" +
                            "<input type='number' class='rating' data-min=0 data-max=10 data-step=0.1 data-size='sm'" +
                            "data-show-clear='false' data-show-caption='false' data-display-only='true' value='"+question.expected+"'/>");
                        // add scores to array
                        var scores = [];
                        for (var userID in question.answers) {
                            if (question.answers.hasOwnProperty(userID)) {
                                scores.push([question.answers[userID], question.answers[userID].score]);
                            }
                        }
                        // sort by score
                        scores.sort(function(a, b) {
                            return b[1] - a[1];
                        });
                        for(var i = 0; i < scores.length ; i++){
                            var attr = "";
                            if(scores[i][0].player = current){
                                attr += "current ";
                            }
                            scoreList.append("<div class='player-wrapper " + attr + "'><span class='place'>" + (i+1) + "</span><span class='name'>" + scores[i][0].player.name + "</span>" +
                                "<span><input type='number' class='rating' data-min=0 data-max=10 data-step=0.1 " +
                                "data-size='xs' data-show-clear='false' data-show-caption='false' " +
                                "data-display-only='true' value='" + scores[i][0].value + "'/></span>" +
                                "<span class='score'>" + scores[i][0].score.toFixed(1) + "</span></div>");
                        }
                        $('.rating').rating('refresh', {});
                        break;
                    case TYPE_CAST:
                        if(question.answers[current.id].score == 10) {
                            popMessage("Alright!");
                            $('#CorrectAudio').trigger("pause").prop("currentTime", 0).trigger("play");
                        }
                        var amswers = [];
                        for (var userID in question.answers) {
                            if (question.answers.hasOwnProperty(userID)) {
                                amswers.push(question.answers[userID]);
                            }
                        }
                        for(var i = 0; i < question.options.length ; i++){
                            var playersVoted = [];
                            for(var j = 0; j < amswers.length ; j++){
                                if(amswers[j].value == i){
                                    playersVoted.push(amswers[j].player)
                                }
                            }
                            var attr = "";
                            if(i == question.expected){
                                attr += "expected "
                            }
                            if(i == question.answers[current.id].value){
                                attr += "current "
                            }
                            scoreList.append("<div class='option-votes " + attr + "'>" +
                                "<img src='https://image.tmdb.org/t/p/w75/" + question.options[i].profile_path + "'/>" +
                                "<div class='title'>" + question.options[i].Name + "</div><div class='votes-sum'>" + playersVoted.length + "</div></div>");
                        }
                        break;
                }
                showFrame(scoreFrame);
                showFrame(movieFrame);
                break;
        }
    }
    if(upd.history != null){
        // history = upd.history;
        hideLoading(gameOverFrame);
        for(var i = 0 ; i < upd.history.length ; i++) {
            var qType;
            var answerBlock;
            switch(upd.history[i].style){
                case TYPE_RATING:
                    qType = "Guess Rating";
                    answerBlock = "<input type='number' class='rating' data-min=0 data-max=10 data-step=0.1" +
                        " data-size='xs' data-show-clear='false' data-show-caption='false'" +
                        "data-display-only='true' value='" + upd.history[i].expected + "'/><div class='rating-label'>" +
                        upd.history[i].expected + "</div>";
                    break;
                case TYPE_CAST:
                    qType = "Who is playing?";
                    answerBlock = "<div class='title'>" + upd.history[i].options[upd.history[i].expected].Name + "</div>" +
                    "<img src='https://image.tmdb.org/t/p/w75/" + upd.history[i].options[upd.history[i].expected].profile_path + "' />";
                    break;
            }
            gameOverFrame.append("<div class='histoy-game'>" +
            "<img src='https://image.tmdb.org/t/p/w75/" + upd.history[i].movie.poster_path + "'/>" +
                "<div class='title'>" + upd.history[i].movie.Title + "</div>" +
                "<div class='expected-answer'><div class='question-type'>" + qType + "</div>" + answerBlock +
                "<div class='current-score'>" + upd.history[i].answers[current.id].score.toFixed(1) + "</div>" +
                "</div></div>");
        }
    }
    if(upd.setup != null){
        hideLoading(setupSelection);
        setup = upd.setup;
        $(".category-selection[value='" + setup.category + "']").prop("checked", true); // NOT WORKING!#!@
        switch (setup.category){
            case CATEGORY_POPULAR:
                setupSelection.empty();
                break;
            case CATEGORY_COMPANY:
                if(setup.options != null) {
                    setupSelection.empty();
                    for (var i = 0; i < setup.options.length; i++) {
                        setupSelection.append("<label class='btn setup-selection'>" +
                            "<input type='radio' name='setup-selection' value='" +
                            setup.options[i].ID + "' autocomplete='off'>" +
                            "<img src='https://image.tmdb.org/t/p/w185/" + setup.options[i].logo_path + "'/>" +
                            "<div class='title'>" + setup.options[i].Name + "</div></label>");
                    }
                }
                break;
            case CATEGORY_GENERE:
                setupSelection.empty();
                for (var i = 0; i < genres.length; i++) {
                    setupSelection.append("<label class='btn setup-selection'>" +
                        "<input type='radio' name='setup-selection'" +
                        "value='" + genres[i].id + "' autocomplete='off'>" +
                        "<div class='title'>" + genres[i].name + "</div></label>");
                }
                break;
        }
    }
    if(upd.ready){

    }
};
ws.onclose = function() {
    scoreFrame.append("<span>connection closed</span>");
};

$('input:radio[name="category"]').change(
    function(){
        if ($(this).is(':checked')) {
            setupCategoryChange($(this).val());
        }
    });

function startGame() {
    ws.send(
        JSON.stringify({
            type: TYPE_START_GAME,
            message: $('input[name="setup-selection"]:checked').val()
        })
    );
    showLoading(setupFrame, "Please Wait, preparing game...");
}
function resumeGame() {
    ws.send(
        JSON.stringify({
            type: TYPE_CONTINUE_GAME
        })
    );
}
function setClock(timeleft){
    timerText.text("");
    clearInterval(timer);
    var angle = 0;
    timer = setInterval(function() {
        if(timeleft < 0) {
            timerText.text("TIME IS UP");
        } else {
            timeleft--;
            timerText.text(timeleft + "s")
            angle = (360 - (timeleft * 6))
            seconds.css("transform", "rotateZ("+ angle + "deg)");
            seconds.css("webkitTransform", "rotateZ("+ angle + "deg)");
            if(timeleft <= 10){
                $('#ClockTickAudio').trigger("pause").prop("currentTime", 0).trigger("play");
            }
        }
    }, 1000);
}
function submitGuess(value){
    ws.send(
        JSON.stringify({
            type: TYPE_PLAYER_ANSWER,
            message: value.toString()
        })
    );
    $('#PingAudio').trigger("pause").prop("currentTime", 0).trigger("play");
}
function updatePlayerList(pl){
    players = pl;
    playersList.empty();
    for (var i = 0; i < players.length ; i++){
        var attr = "player-wrapper ";
        if(players[i].online){
            attr += "online ";
        } else {
            attr += "offline ";
        }
        if(players[i].admin){
            attr += "admin ";
        }
        if(players[i].id != current.id){
            playersList.append("<div class='" + attr + "'><span class='name'>" + players[i].name + "</span><span class='score'>" + players[i].score.toFixed(1) + "</span></div>");
        } else {
            attr += "current ";
            playersList.append("<div class='" + attr + "'><span class='name'>" + current.name + "</span><span class='score'>" + current.score.toFixed(1) + "</span></div>");
        }
    }
}

function popMessage(msg){
    $("#MessagesFrame").append("<div class='awesome'>" + msg + "</div>");
    setTimeout(function () {
        $(".awesome").addClass("active");
        setTimeout(function () {
            $(".awesome").remove();
        }, 500);
    }, 1);
}

function setupCategoryChange(cat){
    ws.send(
        JSON.stringify({
            type: TYPE_SETUP_CATEGORY,
            message: cat.toString()
        })
    );
    showLoading(setupSelection, "Loading, please wait...");
}

function showLoading(frame, msg){
    hideLoading(frame);
    frame.prepend("<div class='loader'><div class='wheel'></div><div class='message'>" + msg + "</div></div>");
}
function hideLoading(frame){
    frame.find(".loader").remove();
}