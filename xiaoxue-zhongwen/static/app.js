const SCENE_ALIAS = {
    house: "island",
    kick: "firstkick",
    run: "daily",
    effort: "daily",
    goodbye: "academy",
    travel: "star",
};

const AVATARS = ["⚽", "🌟", "🐼", "🐶", "🌸", "🔥", "🌈", "🎯"];

const state = {
    child: null,
    children: [],
    course: null,
    progress: null,
    info: null,
    day: 1,
    lesson: null,
    index: 0,
    heard: new Set(),
    seconds: 0,
    timer: null,
    style: localStorage.getItem("learn-style") || "comic",
    voice: localStorage.getItem("learn-voice") || "xiaoxiao",
    voices: [],
    audio: null,
    speakSeq: 0,
    slowRun: 0,
    speakFinish: null,
    avatar: "⚽",
    screen: "kids",
    practice: [],
    practiceIndex: 0,
    game: null,
    gameIndex: 0,
    buildPicked: [],
};

const $ = (id) => document.getElementById(id);

function showScreen(name) {
    state.screen = name;
    document.querySelectorAll(".screen").forEach((screen) => {
        screen.classList.toggle("active", screen.dataset.screen === name);
    });
    $("back-btn").hidden = name === "kids";
    $("child-chip").hidden = !state.child;
    if (state.child) {
        $("child-chip").textContent = `${state.child.avatar} ${state.child.name}`;
    }
    if (name !== "lesson") {
        stopTimer();
        stopSpeak();
    }
}

function sceneUrl(style, scene) {
    const id = SCENE_ALIAS[scene] || scene || "intro";
    return `images/${style}/${id}.webp`;
}

function lineImageUrl(day, line, style) {
    return `/api/image?day=${day}&line=${line}&style=${encodeURIComponent(style)}`;
}

function pickChineseVoice() {
    const voices = window.speechSynthesis ? speechSynthesis.getVoices() : [];
    const want = VoiceById(state.voice);
    const zh = voices.filter((v) => /zh|chinese|中文|普通话|台灣|台湾/i.test(`${v.name} ${v.lang}`));
    if (want.gender === "male") {
        return (
            zh.find((v) => /male|男|yun|kang|kangkang|yunyang|yunxi|yunjian/i.test(`${v.name} ${v.lang}`)) ||
            zh[0] ||
            null
        );
    }
    return (
        zh.find((v) => /female|女|ting|xiao|hui|yao|meijia/i.test(`${v.name} ${v.lang}`)) ||
        zh[0] ||
        null
    );
}

function VoiceById(id) {
    return (state.voices || []).find((v) => v.id === id) || { id: "xiaoxiao", gender: "female", pitch: 1.1, rate: 0.9 };
}

const ttsBlobs = new Map();
const ttsPending = new Map();

function ttsKey(text) {
    return `${state.voice}|${String(text || "").trim()}`;
}

function ttsApiUrl(text) {
    return `/api/tts?voice=${encodeURIComponent(state.voice)}&text=${encodeURIComponent(text)}`;
}

function rememberTts(key, url) {
    if (ttsBlobs.has(key)) ttsBlobs.delete(key);
    ttsBlobs.set(key, url);
    while (ttsBlobs.size > 80) {
        const oldest = ttsBlobs.keys().next().value;
        ttsBlobs.delete(oldest);
    }
}

function fetchTtsBlob(text) {
    text = String(text || "").trim();
    if (!text) return Promise.resolve("");
    const key = ttsKey(text);
    if (ttsBlobs.has(key)) return Promise.resolve(ttsBlobs.get(key));
    if (ttsPending.has(key)) return ttsPending.get(key);
    const pending = fetch(ttsApiUrl(text))
        .then((res) => {
            if (!res.ok) throw new Error("tts failed");
            return res.blob();
        })
        .then((blob) => {
            const url = URL.createObjectURL(blob);
            rememberTts(key, url);
            ttsPending.delete(key);
            return url;
        })
        .catch((err) => {
            ttsPending.delete(key);
            throw err;
        });
    ttsPending.set(key, pending);
    return pending;
}

function prefetchTts(texts) {
    const seen = new Set();
    (texts || []).forEach((text) => {
        text = String(text || "").trim();
        if (!text || seen.has(text)) return;
        seen.add(text);
        fetchTtsBlob(text).catch(() => {});
    });
}

function prefetchLessonAudio() {
    if (!state.lesson) return;
    const beat = currentBeat();
    const texts = [beat.sentence.zh];
    (beat.sentence.tokens || []).forEach((token) => texts.push(token.zh));
    const next = state.lesson.lines[state.index + 1];
    if (next) texts.push(next.sentence.zh);
    prefetchTts(texts);
}

function detachAudio(audio) {
    if (!audio) return;
    audio.onended = null;
    audio.onerror = null;
    audio.oncanplay = null;
    audio.oncanplaythrough = null;
    try {
        audio.pause();
    } catch (_) {}
    try {
        audio.removeAttribute("src");
        audio.load();
    } catch (_) {}
}

function takeSpeakFinish() {
    const fn = state.speakFinish;
    state.speakFinish = null;
    if (fn) fn();
}

function stopSpeak(opts) {
    state.speakSeq += 1;
    if (!opts || !opts.keepSlow) state.slowRun += 1;
    detachAudio(state.audio);
    state.audio = null;
    if (window.speechSynthesis) speechSynthesis.cancel();
    takeSpeakFinish();
}

function speakWeb(text, onend, profile, seq) {
    if (!window.speechSynthesis || !text) {
        if (seq === state.speakSeq && onend) onend();
        return;
    }
    const utter = new SpeechSynthesisUtterance(text);
    const voice = pickChineseVoice();
    utter.lang = voice ? voice.lang : "zh-CN";
    if (voice) utter.voice = voice;
    utter.rate = profile.rate || 0.88;
    utter.pitch = profile.pitch || 1;
    utter.onend = () => {
        if (seq !== state.speakSeq) return;
        if (onend) onend();
    };
    utter.onerror = () => {
        if (seq !== state.speakSeq) return;
        if (onend) onend();
    };
    speechSynthesis.speak(utter);
}

function playAudioSrc(src, text, onend, profile, seq) {
    const audio = new Audio();
    audio.preload = "auto";
    state.audio = audio;
    const alive = () => seq === state.speakSeq && state.audio === audio;
    const done = () => {
        if (!alive()) return;
        detachAudio(audio);
        if (state.audio === audio) state.audio = null;
        if (onend) onend();
    };
    audio.onended = done;
    audio.onerror = () => {
        if (!alive()) return;
        detachAudio(audio);
        if (state.audio === audio) state.audio = null;
        speakWeb(text, onend, profile, seq);
    };
    audio.src = src;
    const tryPlay = () => {
        if (!alive()) return;
        const play = audio.play();
        if (play && play.catch) {
            play.catch(() => {
                if (!alive()) return;
                const retry = () => {
                    if (!alive()) return;
                    audio.play().catch(() => {});
                };
                audio.addEventListener("canplay", retry, { once: true });
            });
        }
    };
    tryPlay();
}

function speakChinese(text, onend, opts) {
    stopSpeak(opts);
    const seq = state.speakSeq;
    const profile = VoiceById(state.voice);
    const finish = () => {
        if (state.speakFinish === finish) state.speakFinish = null;
        if (onend) {
            const cb = onend;
            onend = null;
            cb();
        }
    };
    state.speakFinish = finish;
    text = String(text || "").trim();
    if (!text) {
        finish();
        return;
    }
    const cached = ttsBlobs.get(ttsKey(text));
    playAudioSrc(cached || ttsApiUrl(text), text, finish, profile, seq);
    if (!cached) fetchTtsBlob(text).catch(() => {});
}

function markHeard(index) {
    state.heard.add(index);
    saveProgress({ heard: [...state.heard] });
    renderDots();
}

async function api(url, opts) {
    const res = await fetch(url, opts);
    const data = await res.json().catch(() => ({}));
    if (!res.ok) throw new Error(data.detail || "error");
    return data;
}

async function loadKids() {
    state.children = await api("/api/children");
    state.info = await api("/api/server-info").catch(() => null);
    state.voices = await api("/api/voices").catch(() => []);
    renderKids();
    renderVoices();
}

function renderKids() {
    const grid = $("kid-grid");
    grid.innerHTML = "";
    state.children.forEach((child) => {
        const btn = document.createElement("button");
        btn.type = "button";
        btn.className = "kid-card";
        btn.innerHTML = `<span class="kid-face">${child.avatar}</span><strong>${escapeHtml(child.name)}</strong>`;
        btn.addEventListener("click", () => selectChild(child));
        grid.appendChild(btn);
    });
    if (!state.children.length) {
        grid.innerHTML = `<p class="muted">まだだれもいないよ。したからなまえを入れてね。</p>`;
    }
    $("lan-hint").textContent = state.info
        ? `${state.info.hintJa} ${(state.info.lanUrls || []).join("  ")}`
        : "";
    const row = $("avatar-row");
    row.innerHTML = "";
    AVATARS.forEach((avatar) => {
        const btn = document.createElement("button");
        btn.type = "button";
        btn.className = "avatar-pick" + (avatar === state.avatar ? " active" : "");
        btn.textContent = avatar;
        btn.addEventListener("click", () => {
            state.avatar = avatar;
            renderKids();
        });
        row.appendChild(btn);
    });
}

async function selectChild(child) {
    state.child = child;
    localStorage.setItem("learn-child", child.id);
    state.course = await api("/api/course");
    state.progress = await api(`/api/children/${child.id}/progress`);
    renderCalendar();
    showScreen("calendar");
}

function dayState(day) {
    return (state.progress && state.progress.days && state.progress.days[String(day)]) || null;
}

function isUnlocked(day) {
    if (day <= 1) return true;
    const prev = dayState(day - 1);
    return !!(prev && prev.lessonDone);
}

function renderCalendar() {
    $("calendar-title").textContent = `${state.child.avatar} ${state.child.name} の30日`;
    const done = (state.progress.completedDays || []).length;
    $("calendar-lead").textContent = `クリア ${done} / 30日。毎日15分の物語。終わったらゲームが1つ。`;
    const grid = $("month-grid");
    grid.innerHTML = "";
    state.course.days.forEach((lesson) => {
        const st = dayState(lesson.day);
        const unlocked = isUnlocked(lesson.day);
        const btn = document.createElement("button");
        btn.type = "button";
        btn.className = "day-cell";
        if (!unlocked) btn.classList.add("locked");
        if (st && st.lessonDone) btn.classList.add("done");
        if (st && st.gameDone) btn.classList.add("played");
        btn.disabled = !unlocked;
        btn.innerHTML = `<em>${lesson.day}</em><span>${escapeHtml(lesson.title.ja)}</span>`;
        btn.addEventListener("click", () => openDay(lesson.day));
        grid.appendChild(btn);
    });
}

async function openDay(day) {
    state.day = day;
    state.lesson = await api(`/api/course/day/${day}`);
    const st = dayState(day);
    state.index = 0;
    state.heard = new Set(st && st.heard ? st.heard : []);
    state.seconds = st && st.seconds ? st.seconds : 0;
    renderStyles();
    renderVoices();
    renderLesson();
    showScreen("lesson");
    startTimer();
}

function startTimer() {
    stopTimer();
    updateTimer();
    state.timer = setInterval(() => {
        state.seconds += 1;
        updateTimer();
        if (state.seconds % 20 === 0) saveProgress({ seconds: state.seconds });
    }, 1000);
}

function stopTimer() {
    if (state.timer) clearInterval(state.timer);
    state.timer = null;
}

function updateTimer() {
    const m = String(Math.floor(state.seconds / 60)).padStart(2, "0");
    const s = String(state.seconds % 60).padStart(2, "0");
    $("timer-label").textContent = `${m}:${s}`;
}

function currentBeat() {
    return state.lesson.lines[state.index];
}

function renderStyles() {
    const bar = $("style-bar");
    bar.innerHTML = "";
    ["comic", "picturebook", "realistic"].forEach((id, i) => {
        const ja = ["まんが", "えほん", "リアル"][i];
        const btn = document.createElement("button");
        btn.type = "button";
        btn.className = "style-btn" + (id === state.style ? " active" : "");
        btn.innerHTML = `<img alt="" src="images/${id}/preview.webp"><span>${ja}</span>`;
        btn.addEventListener("click", () => {
            state.style = id;
            localStorage.setItem("learn-style", id);
            renderStyles();
            renderLesson();
        });
        bar.appendChild(btn);
    });
}

function renderVoices() {
    const bar = $("voice-bar");
    if (!bar) return;
    bar.innerHTML = "";
    (state.voices || []).forEach((voice) => {
        const btn = document.createElement("button");
        btn.type = "button";
        btn.className = "voice-btn" + (voice.id === state.voice ? " active" : "");
        btn.textContent = `${voice.gender === "female" ? "♀" : "♂"} ${voice.nameJa}`;
        btn.addEventListener("click", () => {
            state.voice = voice.id;
            localStorage.setItem("learn-voice", voice.id);
            renderVoices();
            prefetchLessonAudio();
            speakChinese("你好，我是" + voice.nameZh);
        });
        bar.appendChild(btn);
    });
}

function renderLesson() {
    const beat = currentBeat();
    const lesson = state.lesson;
    $("day-moral").textContent = `${lesson.title.ja}　${lesson.moral.ja}`;
    const img = $("scene-image");
    const frame = $("picture-frame");
    const loading = $("picture-loading");
    const src = lineImageUrl(state.day, state.index, state.style);
    if (frame) frame.classList.add("loading");
    if (loading) loading.hidden = false;
    img.onload = () => {
        if (frame) frame.classList.remove("loading");
        if (loading) loading.hidden = true;
    };
    img.onerror = () => {
        img.onerror = null;
        img.src = sceneUrl(state.style, beat.scene || lesson.scene);
        if (loading) loading.hidden = true;
        if (frame) frame.classList.remove("loading");
    };
    img.src = src;
    img.alt = beat.sentence.zh;
    $("scene-caption").textContent = `${lesson.day}日目 · ${state.index + 1} / ${lesson.lines.length}　${beat.sentence.zh}`;
    if (state.index + 1 < lesson.lines.length) {
        const preload = new Image();
        preload.src = lineImageUrl(state.day, state.index + 1, state.style);
    }
    $("pattern-card").innerHTML = `<div class="pattern-label">句型</div>
        <p class="pattern-zh">${escapeHtml(beat.pattern.zh)}</p>
        <p class="pattern-ja">${escapeHtml(beat.pattern.ja)}</p>`;
    $("sentence-pinyin").textContent = beat.sentence.pinyin;
    $("sentence-ja").textContent = beat.sentence.ja;
    $("sentence-words").innerHTML = "";
    (beat.sentence.tokens || []).forEach((token) => {
        const btn = document.createElement("button");
        btn.type = "button";
        btn.className = "word";
        btn.textContent = token.zh;
        btn.addEventListener("click", (event) => {
            event.stopPropagation();
            speakChinese(token.zh);
            markHeard(state.index);
        });
        $("sentence-words").appendChild(btn);
    });
    renderDots();
    $("prev-btn").disabled = state.index === 0;
    const last = state.index === lesson.lines.length - 1;
    const allHeard = lesson.lines.every((_, i) => state.heard.has(i));
    $("next-btn").textContent = last && allHeard ? "15分クリアへ" : last ? "ぜんぶ聞こう" : "つぎの文";
    prefetchLessonAudio();
}

function renderDots() {
    const path = $("progress-path");
    path.innerHTML = "";
    state.lesson.lines.forEach((_, i) => {
        const dot = document.createElement("button");
        dot.type = "button";
        dot.className = "dot";
        if (i === state.index) dot.classList.add("current");
        if (state.heard.has(i)) dot.classList.add("done");
        dot.addEventListener("click", () => {
            state.index = i;
            renderLesson();
        });
        path.appendChild(dot);
    });
}

function go(delta) {
    const next = state.index + delta;
    if (next < 0) return;
    if (next >= state.lesson.lines.length) {
        const allHeard = state.lesson.lines.every((_, i) => state.heard.has(i));
        if (!allHeard) {
            speakChinese("再听一听。");
            return;
        }
        openPractice();
        return;
    }
    state.index = next;
    renderLesson();
}

function contentWords(limit) {
    const skip = new Set(["他", "的", "了", "很", "在", "是", "也", "都", "不", "有", "和"]);
    const seen = new Set();
    const out = [];
    state.lesson.lines.forEach((beat) => {
        (beat.sentence.tokens || []).forEach((token) => {
            if (skip.has(token.zh) || seen.has(token.zh)) return;
            seen.add(token.zh);
            out.push(token);
        });
    });
    return out.slice(0, limit);
}

function openPractice() {
    stopTimer();
    const words = contentWords(4);
    const pool = contentWords(12);
    state.practice = words.map((word, i) => {
        const options = [word.zh];
        pool.forEach((other) => {
            if (other.zh !== word.zh && options.length < 4) options.push(other.zh);
        });
        return { word, options, answer: word.zh };
    });
    state.practiceIndex = 0;
    renderPractice();
    showScreen("practice");
}

function renderPractice() {
    const item = state.practice[state.practiceIndex];
    const card = $("practice-card");
    if (!item) {
        finishLesson();
        return;
    }
    card.innerHTML = `<p class="practice-q">${state.practiceIndex + 1} / ${state.practice.length}</p>
        <button type="button" class="primary-btn big" id="practice-play">▶ 中国語を聞く</button>
        <p class="lead">${escapeHtml(item.word.ja)} はどれ？</p>
        <div class="choice-grid" id="practice-choices"></div>`;
    $("practice-play").addEventListener("click", () => speakChinese(item.word.zh));
    speakChinese(item.word.zh);
    const grid = $("practice-choices");
    item.options.forEach((opt) => {
        const btn = document.createElement("button");
        btn.type = "button";
        btn.className = "choice-btn";
        btn.textContent = opt;
        btn.addEventListener("click", () => {
            if (opt === item.answer) {
                state.practiceIndex += 1;
                renderPractice();
            } else {
                btn.classList.add("wrong");
                speakChinese(item.word.zh);
            }
        });
        grid.appendChild(btn);
    });
}

async function finishLesson() {
    await saveProgress({
        heard: [...state.heard],
        lessonDone: true,
        seconds: state.seconds,
    });
    $("reward-title").textContent = `15分できた！ ${state.child.name}`;
    $("reward-moral").textContent = state.lesson.moral.ja;
    showScreen("reward");
}

async function saveProgress(extra) {
    if (!state.child || !state.lesson) return;
    const patch = {
        day: state.day,
        seconds: state.seconds,
        ...extra,
    };
    try {
        state.progress = await api(`/api/children/${state.child.id}/progress`, {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify(patch),
        });
    } catch (err) {
        console.error(err);
    }
}

async function openGame() {
    try {
        state.game = await api(`/api/game/${state.day}?child=${encodeURIComponent(state.child.id)}`);
    } catch (err) {
        $("game-hint").textContent = err.message;
        showScreen("game");
        return;
    }
    state.gameIndex = 0;
    state.buildPicked = [];
    $("game-title").textContent = state.game.title.ja;
    $("game-hint").textContent = state.game.hint.ja;
    renderGame();
    showScreen("game");
}

function renderGame() {
    const board = $("game-board");
    const item = state.game.items[state.gameIndex];
    if (!item) {
        board.innerHTML = `<p class="stamp">🎉</p><p class="lead">ゲームクリア！</p>
            <button type="button" class="primary-btn" id="game-done">カレンダーへ</button>`;
        $("game-done").addEventListener("click", async () => {
            await saveProgress({ lessonDone: true, gameDone: true });
            renderCalendar();
            showScreen("calendar");
        });
        return;
    }
    if (state.game.kind === "build") {
        renderBuild(board, item);
        return;
    }
    board.innerHTML = `<p class="practice-q">${state.gameIndex + 1} / ${state.game.items.length}</p>
        <p class="game-prompt">${escapeHtml(item.prompt.zh)}</p>
        <p class="muted">${escapeHtml(item.prompt.ja)}</p>
        <button type="button" class="nav-btn" id="game-play">▶ 聞く</button>
        <div class="choice-grid" id="game-choices"></div>`;
    $("game-play").addEventListener("click", () => speakChinese(item.audioZh || item.prompt.zh));
    speakChinese(item.audioZh || item.prompt.zh);
    const grid = $("game-choices");
    item.options.forEach((opt, i) => {
        const btn = document.createElement("button");
        btn.type = "button";
        btn.className = "choice-btn";
        btn.textContent = opt;
        btn.addEventListener("click", () => {
            if (i === item.answer) {
                state.gameIndex += 1;
                renderGame();
            } else {
                btn.classList.add("wrong");
                speakChinese(item.audioZh || item.prompt.zh);
            }
        });
        grid.appendChild(btn);
    });
}

function renderBuild(board, item) {
    const picked = state.buildPicked;
    const target = (item.tokens || []).map((t) => t.zh).join("");
    board.innerHTML = `<p class="practice-q">${state.gameIndex + 1} / ${state.game.items.length}</p>
        <button type="button" class="nav-btn" id="game-play">▶ 文を聞く</button>
        <p class="build-line" id="build-line">${picked.join(" ") || "……"}</p>
        <div class="choice-grid" id="game-choices"></div>`;
    $("game-play").addEventListener("click", () => speakChinese(item.audioZh));
    const grid = $("game-choices");
    item.options.forEach((opt) => {
        const used = picked.filter((p) => p === opt).length;
        const total = item.options.filter((p) => p === opt).length;
        const btn = document.createElement("button");
        btn.type = "button";
        btn.className = "choice-btn";
        btn.textContent = opt;
        if (used >= total) btn.disabled = true;
        btn.addEventListener("click", () => {
            state.buildPicked.push(opt);
            if (state.buildPicked.join("") === target) {
                state.gameIndex += 1;
                state.buildPicked = [];
                renderGame();
                return;
            }
            if (state.buildPicked.join("").length >= target.length && state.buildPicked.join("") !== target) {
                state.buildPicked = [];
            }
            renderBuild(board, item);
        });
        grid.appendChild(btn);
    });
}

function escapeHtml(value) {
    return String(value || "")
        .replace(/&/g, "&amp;")
        .replace(/</g, "&lt;")
        .replace(/>/g, "&gt;")
        .replace(/"/g, "&quot;");
}

$("add-kid").addEventListener("submit", async (event) => {
    event.preventDefault();
    const name = $("kid-name").value.trim();
    if (!name) return;
    const child = await api("/api/children", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ name, avatar: state.avatar }),
    });
    $("kid-name").value = "";
    await loadKids();
    selectChild(child);
});

$("back-btn").addEventListener("click", () => {
    if (state.screen === "lesson" || state.screen === "practice" || state.screen === "reward" || state.screen === "game") {
        stopTimer();
        saveProgress({ seconds: state.seconds, heard: [...state.heard] });
        renderCalendar();
        showScreen("calendar");
        return;
    }
    showScreen("kids");
});

$("child-chip").addEventListener("click", () => showScreen("kids"));
$("sentence-card").addEventListener("click", () => {
    if (!state.lesson) return;
    speakChinese(currentBeat().sentence.zh);
    markHeard(state.index);
});
$("prev-btn").addEventListener("click", () => go(-1));
$("next-btn").addEventListener("click", () => go(1));
$("slow-btn").addEventListener("click", async () => {
    const beat = currentBeat();
    const runId = state.slowRun + 1;
    state.slowRun = runId;
    for (const token of beat.sentence.tokens || []) {
        if (runId !== state.slowRun) return;
        await new Promise((resolve) => speakChinese(token.zh, resolve, { keepSlow: true }));
        if (runId !== state.slowRun) return;
    }
    if (runId === state.slowRun) markHeard(state.index);
});
$("open-game-btn").addEventListener("click", openGame);
$("back-calendar-btn").addEventListener("click", () => {
    renderCalendar();
    showScreen("calendar");
});

document.addEventListener("keydown", (event) => {
    if (state.screen !== "lesson") return;
    if (event.key === "ArrowRight") go(1);
    if (event.key === "ArrowLeft") go(-1);
});

if (window.speechSynthesis) {
    speechSynthesis.getVoices();
    speechSynthesis.addEventListener("voiceschanged", () => pickChineseVoice());
}

loadKids()
    .then(async () => {
        const last = localStorage.getItem("learn-child");
        const found = state.children.find((c) => c.id === last);
        if (found) selectChild(found);
    })
    .catch(() => {
        $("lan-hint").textContent = "サーバーに接続できなかったよ。xiaoxue-zhongwen で go run . してね。";
    });
