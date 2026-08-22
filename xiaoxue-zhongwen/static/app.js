const SCENE_ALIAS = {
    house: "island",
    kick: "firstkick",
    run: "daily",
    effort: "daily",
    goodbye: "academy",
    travel: "star",
};

const state = {
    catalog: null,
    match: null,
    books: [],
    story: null,
    index: 0,
    style: localStorage.getItem("learn-style") || "comic",
    showZhHint: localStorage.getItem("learn-zh-hint") === "1",
    screen: "home",
};

const els = {
    screens: [...document.querySelectorAll(".screen")],
    backBtn: document.getElementById("back-btn"),
    langHintBtn: document.getElementById("lang-hint-btn"),
    interestGrid: document.getElementById("interest-grid"),
    askForm: document.getElementById("ask-form"),
    interestInput: document.getElementById("interest-input"),
    matchBubble: document.getElementById("match-bubble"),
    matchBubbleZh: document.getElementById("match-bubble-zh"),
    storyList: document.getElementById("story-list"),
    bookList: document.getElementById("book-list"),
    styleBar: document.getElementById("style-bar"),
    sceneImage: document.getElementById("scene-image"),
    sceneCaption: document.getElementById("scene-caption"),
    patternCard: document.getElementById("pattern-card"),
    sentenceCard: document.getElementById("sentence-card"),
    sentencePinyin: document.getElementById("sentence-pinyin"),
    sentenceWords: document.getElementById("sentence-words"),
    sentenceJa: document.getElementById("sentence-ja"),
    progressPath: document.getElementById("progress-path"),
    prevBtn: document.getElementById("prev-btn"),
    nextBtn: document.getElementById("next-btn"),
    slowBtn: document.getElementById("slow-btn"),
};

function showScreen(name) {
    state.screen = name;
    els.screens.forEach((screen) => {
        screen.classList.toggle("active", screen.dataset.screen === name);
    });
    els.backBtn.hidden = name === "home";
}

function sceneUrl(style, scene) {
    const id = SCENE_ALIAS[scene] || scene || "intro";
    return `images/${style}/${id}.webp`;
}

function pickChineseVoice() {
    const voices = window.speechSynthesis ? speechSynthesis.getVoices() : [];
    return (
        voices.find((v) => /^zh(-|$)/i.test(v.lang.replace("_", "-"))) ||
        voices.find((v) => /chinese|中文|普通话|台灣|台湾/i.test(`${v.name} ${v.lang}`)) ||
        null
    );
}

function speakChinese(text, onend) {
    if (!window.speechSynthesis || !text) {
        if (onend) onend();
        return;
    }
    speechSynthesis.cancel();
    const utter = new SpeechSynthesisUtterance(text);
    const voice = pickChineseVoice();
    utter.lang = voice ? voice.lang : "zh-CN";
    if (voice) utter.voice = voice;
    utter.rate = 0.85;
    utter.pitch = 1;
    utter.onend = () => onend && onend();
    utter.onerror = () => onend && onend();
    speechSynthesis.speak(utter);
}

function highlightWord(index) {
    els.sentenceWords.querySelectorAll(".word").forEach((btn, i) => {
        btn.classList.toggle("speaking", i === index);
    });
}

function speakWord(token, index) {
    highlightWord(index);
    speakChinese(token.zh, () => highlightWord(-1));
}

function speakSentence(beat) {
    els.sentenceCard.classList.add("speaking");
    speakChinese(beat.sentence.zh, () => els.sentenceCard.classList.remove("speaking"));
}

async function speakSlow(beat) {
    const tokens = beat.sentence.tokens || [];
    for (let i = 0; i < tokens.length; i += 1) {
        highlightWord(i);
        await new Promise((resolve) => speakChinese(tokens[i].zh, resolve));
    }
    highlightWord(-1);
}

async function loadCatalog() {
    const res = await fetch("/api/catalog");
    if (!res.ok) throw new Error("catalog failed");
    state.catalog = await res.json();
    renderInterests();
    renderStyles();
}

function renderInterests() {
    els.interestGrid.innerHTML = "";
    (state.catalog.interests || []).forEach((interest) => {
        const btn = document.createElement("button");
        btn.type = "button";
        btn.className = "interest-chip";
        btn.textContent = `${interest.emoji} ${interest.ja}`;
        btn.addEventListener("click", () => ask(interest.ja));
        els.interestGrid.appendChild(btn);
    });
}

function renderStyles() {
    els.styleBar.innerHTML = "";
    (state.catalog.styles || []).forEach((style) => {
        const btn = document.createElement("button");
        btn.type = "button";
        btn.className = "style-btn" + (style.id === state.style ? " active" : "");
        btn.setAttribute("role", "tab");
        btn.setAttribute("aria-selected", style.id === state.style ? "true" : "false");
        btn.innerHTML = `<img alt="" src="images/${style.id}/preview.webp"><span>${style.ja}</span>`;
        btn.addEventListener("click", () => {
            state.style = style.id;
            localStorage.setItem("learn-style", style.id);
            renderStyles();
            renderReader();
        });
        els.styleBar.appendChild(btn);
    });
}

async function ask(query) {
    const text = (query || "").trim();
    if (!text) return;
    els.interestInput.value = text;
    const res = await fetch("/api/interest", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ text }),
    });
    if (!res.ok) {
        els.matchBubble.textContent = "いまは話を探せなかったよ。もういちど試してね。";
        showScreen("match");
        return;
    }
    state.match = await res.json();
    renderMatch();
    showScreen("match");
    loadBooks(state.match.bookQuery || text);
}

function renderMatch() {
    const match = state.match;
    els.matchBubble.textContent = match.messageJa;
    els.matchBubbleZh.textContent = match.messageZh;
    els.matchBubbleZh.hidden = !state.showZhHint;
    els.storyList.innerHTML = "";
    (match.stories || []).forEach((story) => {
        const btn = document.createElement("button");
        btn.type = "button";
        btn.className = "story-card";
        btn.innerHTML = `<h3>${escapeHtml(story.title.ja)}</h3>
            <p>${escapeHtml(story.summary.ja)}</p>
            <p class="muted">${story.beats.length} 文 · ${escapeHtml(story.title.zh)}</p>`;
        btn.addEventListener("click", () => openStory(story));
        els.storyList.appendChild(btn);
    });
}

async function loadBooks(query) {
    els.bookList.innerHTML = `<p class="muted">本を探しています…</p>`;
    try {
        const res = await fetch(`/api/books?q=${encodeURIComponent(query)}`);
        const data = await res.json();
        state.books = data.books || [];
        if (!state.books.length) {
            els.bookList.innerHTML = `<p class="muted">${data.noteJa || "関連する本はまだないよ。"}</p>`;
            return;
        }
        els.bookList.innerHTML = "";
        state.books.forEach((book) => {
            const card = document.createElement(book.infoUrl ? "a" : "div");
            card.className = "book-card";
            if (book.infoUrl) {
                card.href = book.infoUrl;
                card.target = "_blank";
                card.rel = "noopener noreferrer";
            }
            const cover = book.coverUrl
                ? `<img src="${escapeAttr(book.coverUrl)}" alt="">`
                : `<img alt="" src="images/${state.style}/book.webp">`;
            card.innerHTML = `${cover}<div>
                <h3>${escapeHtml(book.title)}</h3>
                <p>${escapeHtml(book.authors || "")}</p>
                <p class="muted">${escapeHtml(book.noteJa || "")}</p>
            </div>`;
            els.bookList.appendChild(card);
        });
    } catch (err) {
        els.bookList.innerHTML = `<p class="muted">本の検索はいま使えないよ。Cロの子ども向け伝記を、保護者の人と探してみてね。</p>`;
    }
}

function openStory(story) {
    state.story = story;
    state.index = 0;
    renderStyles();
    renderReader();
    showScreen("reader");
}

function currentBeat() {
    return state.story.beats[state.index];
}

function renderReader() {
    if (!state.story) return;
    const beat = currentBeat();
    const img = sceneUrl(state.style, beat.scene);
    els.sceneImage.onerror = () => {
        els.sceneImage.onerror = null;
        els.sceneImage.src = sceneUrl(state.style, "intro");
    };
    els.sceneImage.src = img;
    els.sceneImage.alt = `${state.story.title.ja} ${state.index + 1}文目の絵`;
    els.sceneCaption.textContent = `${state.story.title.ja} · ${state.index + 1} / ${state.story.beats.length}`;

    els.patternCard.innerHTML = `
        <div class="pattern-label">句型 PATTERN</div>
        <p class="pattern-zh">${escapeHtml(beat.pattern.zh)}</p>
        <p class="pattern-ja">${escapeHtml(beat.pattern.ja)}</p>`;

    els.sentencePinyin.textContent = beat.sentence.pinyin;
    els.sentenceJa.textContent = beat.sentence.ja;
    els.sentenceJa.hidden = false;
    els.sentenceWords.innerHTML = "";
    (beat.sentence.tokens || []).forEach((token, index) => {
        const btn = document.createElement("button");
        btn.type = "button";
        btn.className = "word";
        btn.textContent = token.zh;
        btn.title = `${token.pinyin} ${token.ja}`.trim();
        btn.setAttribute("aria-label", `${token.zh} ${token.pinyin || ""} を聞く`);
        btn.addEventListener("click", (event) => {
            event.stopPropagation();
            speakWord(token, index);
        });
        els.sentenceWords.appendChild(btn);
    });

    els.progressPath.innerHTML = "";
    state.story.beats.forEach((_, i) => {
        const dot = document.createElement("button");
        dot.type = "button";
        dot.className = "dot" + (i === state.index ? " current" : i < state.index ? " done" : "");
        dot.setAttribute("aria-label", `${i + 1}文目へ`);
        dot.addEventListener("click", () => {
            state.index = i;
            renderReader();
        });
        els.progressPath.appendChild(dot);
    });

    els.prevBtn.disabled = state.index === 0;
    els.nextBtn.textContent = state.index === state.story.beats.length - 1 ? "おわり" : "つぎの文";
}

function go(delta) {
    if (!state.story) return;
    const next = state.index + delta;
    if (next < 0) return;
    if (next >= state.story.beats.length) {
        showScreen("match");
        return;
    }
    state.index = next;
    renderReader();
}

function escapeHtml(value) {
    return String(value || "")
        .replace(/&/g, "&amp;")
        .replace(/</g, "&lt;")
        .replace(/>/g, "&gt;")
        .replace(/"/g, "&quot;");
}

function escapeAttr(value) {
    return escapeHtml(value).replace(/'/g, "&#39;");
}

els.askForm.addEventListener("submit", (event) => {
    event.preventDefault();
    ask(els.interestInput.value);
});

els.backBtn.addEventListener("click", () => {
    if (state.screen === "reader") showScreen("match");
    else showScreen("home");
});

els.langHintBtn.addEventListener("click", () => {
    state.showZhHint = !state.showZhHint;
    localStorage.setItem("learn-zh-hint", state.showZhHint ? "1" : "0");
    els.langHintBtn.textContent = state.showZhHint ? "中文提示" : "日本語";
    if (state.match) renderMatch();
});

els.sentenceCard.addEventListener("click", () => {
    if (state.story) speakSentence(currentBeat());
});

els.prevBtn.addEventListener("click", () => go(-1));
els.nextBtn.addEventListener("click", () => go(1));
els.slowBtn.addEventListener("click", () => {
    if (state.story) speakSlow(currentBeat());
});

document.addEventListener("keydown", (event) => {
    if (state.screen !== "reader") return;
    if (event.key === "ArrowRight") go(1);
    if (event.key === "ArrowLeft") go(-1);
});

if (window.speechSynthesis) {
    speechSynthesis.getVoices();
    speechSynthesis.addEventListener("voiceschanged", () => pickChineseVoice());
}

els.langHintBtn.textContent = state.showZhHint ? "中文提示" : "日本語";

loadCatalog().catch(() => {
    document.getElementById("home-bubble").textContent =
        "サーバーに接続できなかったよ。xiaoxue-zhongwen で go run . してから、もう一度開いてね。";
});
