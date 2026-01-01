const state = {
    offsetID: 0,
    addOffset: 0,
    isLoading: false,
    selected: new Set(),
    hasMore: true,
    limit: 20,
    total: 0,
    userID: 0,
    sortOrder: 'desc' // 'desc' (Newest first) or 'asc' (Oldest first)
};

const dom = {
    grid: document.getElementById('message-grid'),
    loadMoreBtn: document.getElementById('load-more-btn'),
    deleteBtn: document.getElementById('delete-btn'),
    selectionCount: document.getElementById('selection-count'),
    loader: document.getElementById('loader'),
    totalCount: document.getElementById('total-count'),
    limitSelect: document.getElementById('limit-select'),
    selectEmptyBtn: document.getElementById('select-empty-btn'),
    newestBtn: document.getElementById('newest-btn'),
    oldestBtn: document.getElementById('oldest-btn')
};

function logAction(message) {
    console.log(`[UI LOG] ${message}`);
}

function linkify(text) {
    if (!text) return text;
    // Basic URL regex
    const urlRegex = /(https?:\/\/[^\s]+)/g;
    return text.replace(urlRegex, (url) => `<a href="${url}" target="_blank" rel="noopener noreferrer" style="color: var(--accent); text-decoration: underline;">${url}</a>`);
}

async function fetchMessages(opts = {}) {
    if (state.isLoading) return;

    // If resetting (Newest/Oldest/LimitChange), clear grid and state
    if (opts.reset) {
        state.offsetID = opts.offsetID !== undefined ? opts.offsetID : 0;
        state.addOffset = opts.addOffset || 0;
        state.hasMore = true;
        state.selected.clear();
        dom.grid.innerHTML = '';
        dom.loadMoreBtn.style.display = 'block';
    }

    if (!state.hasMore) return;

    state.isLoading = true;
    dom.loadMoreBtn.disabled = true;
    dom.loadMoreBtn.textContent = "Loading...";

    const fetchLimit = opts.limit || state.limit;

    try {
        logAction(`Fetching messages (limit: ${fetchLimit}, offset: ${state.offsetID}, add_offset: ${state.addOffset})...`);
        const res = await fetch(`/api/messages?limit=${fetchLimit}&offset_id=${state.offsetID}&add_offset=${state.addOffset}&_t=${Date.now()}`);
        if (!res.ok) throw new Error('Failed to fetch');

        const data = await res.json();

        // Update UserID if present
        if (data.user_id) state.userID = data.user_id;

        // Update Total Count
        if (data.total !== undefined) {
            state.total = data.total;
            if (dom.totalCount) dom.totalCount.textContent = data.total;
        }

        if (!data.messages || data.messages.length === 0) {
            state.hasMore = false;
            dom.loadMoreBtn.style.display = 'none';
            logAction("No more messages found.");
            return;
        }

        let messages = data.messages;

        // If Ascending (Oldest First), we reverse the batch
        if (state.sortOrder === 'asc') {
            messages.reverse();
        }

        renderMessages(messages);
        logAction(`Loaded ${messages.length} messages.`);

        // Update offset for next page
        const lastMsg = messages[messages.length - 1];

        if (state.sortOrder === 'asc') {
            // In Ascending mode, lastMsg is the NEWEST of the batch.
            // We want to load even newer messages.
            // To get newer messages relative to lastMsg (going up in ID):
            // We use offset_id = lastMsg.id, and add_offset = -limit.
            state.offsetID = lastMsg.id;
            state.addOffset = -(state.limit);
        } else {
            // Standard Descending mode (New -> Old)
            state.offsetID = lastMsg.id;
            state.addOffset = 0;
        }

    } catch (err) {
        console.error(err);
        alert('Error loading messages');
    } finally {
        state.isLoading = false;
        dom.loadMoreBtn.disabled = false;
        dom.loadMoreBtn.textContent = "Load More";
        updateUI();
    }
}

function renderMessages(messages) {
    messages.forEach(msg => {
        const card = document.createElement('div');
        card.className = 'message-card';
        card.dataset.id = msg.id;

        const dateStr = new Date(msg.date * 1000).toLocaleString();

        let contentHtml = '';
        if (msg.message && msg.message.trim().length > 0) {
            contentHtml = linkify(msg.message);
        } else {
            contentHtml = '<i>(No text content)</i>';
            card.classList.add('is-empty');
        }

        // ... (Media Logic unchanged, but re-include for Replace) ...
        let mediaHtml = '';
        if (msg.attachments && msg.attachments.length > 0) {
            mediaHtml = '<div class="media-grid" style="display: flex; gap: 8px; flex-wrap: wrap; margin-bottom: 8px;">';
            msg.attachments.forEach(att => {
                if (att.type === "Photo") {
                    mediaHtml += `<img src="/api/media?id=${att.id}" loading="lazy" alt="Photo ${att.id}" style="max-height: 200px; max-width: 100%; border-radius: 4px;">`;
                } else {
                    mediaHtml += `<div class="media-tag">${att.type}</div>`;
                }
            });
            mediaHtml += '</div>';
        } else if (msg.media_type) {
            if (msg.media_type === "WebLink" && msg.web_preview) {
            } else {
                if (msg.media_type === "Photo") {
                    mediaHtml = `<div style="margin-bottom: 8px;"><img src="/api/media?id=${msg.id}" loading="lazy" alt="Photo ${msg.id}"></div>`;
                } else {
                    mediaHtml = `<span class="media-tag" style="margin-bottom: 8px; display:inline-block;">${msg.media_type}</span>`;
                }
            }
        }

        // ... (Preview Logic unchanged) ...
        let previewHtml = '';
        if (msg.web_preview) {
            previewHtml = `
            <div class="web-preview" style="border-left: 3px solid var(--accent); padding-left: 8px; margin-top: 8px; background: #2a2a2a; padding: 8px; border-radius: 4px;">
                <div style="font-weight: bold; font-size: 13px; color: var(--accent);">${msg.web_preview.site_name || 'Link'}</div>
                <div style="font-weight: 600; margin-bottom: 4px;"><a href="${msg.web_preview.url}" target="_blank" style="color: inherit; text-decoration: none;">${msg.web_preview.title || msg.web_preview.url}</a></div>
                <div style="font-size: 12px; color: var(--text-secondary);">${msg.web_preview.description || ''}</div>
                ${msg.media_type === 'WebLink' ? `<div style="margin-top:4px;"><img src="/api/media?id=${msg.id}" style="max-height: 150px; border-radius: 4px; display: block;" loading="lazy" onerror="this.style.display='none'"></div>` : ''}
            </div>
            `;
        }

        card.dataset.ids = JSON.stringify(msg.ids || [msg.id]);

        // Deep Link for ID
        // Direct message linking is not supported for Saved Messages.
        // We link to the chat and copy ID on click.
        const idLink = state.userID ? `tg://user?id=${state.userID}` : '#';

        card.innerHTML = `
            <div class="checkbox-wrapper">
                <input type="checkbox" value="${msg.id}"> 
            </div>
            <div class="meta">
                <span><a href="${idLink}" class="id-link" title="Open Telegram & Copy ID" style="color: inherit; text-decoration: none; border-bottom: 1px dashed var(--text-secondary);">ID: ${msg.id}</a> ${msg.ids && msg.ids.length > 1 ? `(+${msg.ids.length - 1})` : ''}</span>
                <span>${dateStr}</span>
            </div>
            ${mediaHtml}
            <div class="content">
                ${contentHtml}
            </div>
            ${previewHtml}
        `;

        const checkbox = card.querySelector('input');
        const allIds = msg.ids || [msg.id];
        const idAnchor = card.querySelector('.id-link');

        checkbox.addEventListener('change', (e) => toggleSelection(allIds, e.target.checked));

        // Click handler logic needs update to ignore ID link click
        card.addEventListener('click', (e) => {
            // If click is on checkbox, image, OR link (anchor tag), ignore selection toggle
            if (e.target !== checkbox && e.target.tagName !== 'IMG' && e.target.tagName !== 'A' && !e.target.closest('a')) {
                checkbox.checked = !checkbox.checked;
                toggleSelection(allIds, checkbox.checked);
            }
        });

        // Smart Link: Copy ID + Open Telegram
        if (idAnchor) {
            idAnchor.addEventListener('click', (e) => {
                e.stopPropagation();

                // Copy ID to clipboard
                navigator.clipboard.writeText(msg.id.toString()).then(() => {
                    // We could show a toast, but for now simple log or let it be.
                    // A blocking alert is annoying but ensures user knows.
                    // Let's use a temporary tooltip change or just rely on the user knowing.
                    // The user asked "opens telegram instead...".
                    // If we give them the ID, they can search.
                    // Let's alert briefly or just let it happen.
                    logAction("ID " + msg.id + " copied to clipboard.");
                });
            });
        }

        dom.grid.appendChild(card);
    });
}
// ... (Logic for toggle/delete unchanged) ...

function handleNewest() {
    state.sortOrder = 'desc';
    fetchMessages({ reset: true, addOffset: 0, offsetID: 0 });
}

async function handleOldest() {
    state.sortOrder = 'asc';

    if (state.total === 0) {
        // Safe-guard: If total is unknown, fetch newest first (limit 1) to populate it
        await fetchMessages({ limit: 1, reset: false });
        // Note: reset: false to avoid clearing grid unnecessarily, 
        // but we are about to jump anyway.
    }

    const total = state.total;
    const limit = state.limit;

    // Calculate offset to reach end.
    // Use Math.max to avoid negative offset
    const addOffset = Math.max(0, total - limit);

    // Explicitly set offsetID to 0 (Start of list) + addOffset (Skip to end)
    await fetchMessages({ reset: true, addOffset: addOffset, offsetID: 0 });
}

dom.loadMoreBtn.addEventListener('click', () => fetchMessages());
dom.newestBtn.addEventListener('click', handleNewest);
dom.oldestBtn.addEventListener('click', handleOldest);
// ... (Existing listeners) ...
dom.deleteBtn.addEventListener('click', deleteSelected);
dom.selectEmptyBtn.addEventListener('click', selectEmpty);
dom.limitSelect.addEventListener('change', handleLimitChange);

// Initial Load
fetchMessages();

// Accepts array of IDs
function toggleSelection(ids, isSelected) {
    if (!Array.isArray(ids)) ids = [ids];

    // We need to find the card element. We assume the first ID is the main ID used for data-id
    // But wait, if we select by list, we update state.
    // Visual selection uses the main ID card.

    const mainID = ids[0];
    const card = document.querySelector(`.message-card[data-id="${mainID}"]`);

    ids.forEach(id => {
        if (isSelected) {
            state.selected.add(id);
        } else {
            state.selected.delete(id);
        }
    });

    // Visual update
    if (card) {
        if (isSelected) card.classList.add('selected');
        else card.classList.remove('selected');
    }

    // Also update checkbox if function called programmatically
    if (card) {
        const cb = card.querySelector('input');
        if (cb) cb.checked = isSelected;
    }

    updateUI();
}

function updateUI() {
    dom.selectionCount.textContent = `${state.selected.size} selected`;
    dom.deleteBtn.disabled = state.selected.size === 0;
}

async function deleteSelected() {
    if (!state.selected.size) return;
    if (!confirm(`Delete ${state.selected.size} messages?`)) return;

    const ids = Array.from(state.selected);
    logAction(`Deleting ${ids.length} messages...`);

    try {
        const res = await fetch('/api/delete', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ ids })
        });

        if (!res.ok) throw new Error('Delete failed');

        // Remove from UI
        document.querySelectorAll('.message-card').forEach(card => {
            const mainId = parseInt(card.dataset.id);
            // We only need to match the Main ID of the card to one of the deleted IDs
            if (ids.includes(mainId)) {
                card.remove();
            }
        });

        // Update State
        const deletedCount = ids.length; // Approximate, assuming all successful
        state.total = Math.max(0, state.total - deletedCount);
        if (dom.totalCount) dom.totalCount.textContent = state.total;

        state.selected.clear();
        updateUI();
        logAction(`Successfully deleted ${deletedCount} messages.`);

        // If grid is empty after delete, maybe try to fetch more?
        if (dom.grid.children.length === 0 && state.hasMore) {
            fetchMessages();
        }

    } catch (err) {
        console.error(err);
        alert('Failed to delete messages');
    }
}

function selectEmpty() {
    const cards = document.querySelectorAll('.message-card.is-empty');
    let addedCount = 0;
    console.log('[DEBUG] Found', cards.length, 'empty message cards');

    cards.forEach(card => {
        const checkbox = card.querySelector('input');
        if (!checkbox.checked) {
            checkbox.checked = true;
            // Parse ID safely
            const id = parseInt(card.dataset.id);
            if (!isNaN(id)) {
                toggleSelection(id, true);
                addedCount++;
            }
        }
    });

    if (addedCount === 0) {
        if (cards.length === 0) {
            alert("No visible empty messages found.");
        } else {
            alert("All empty messages are already selected.");
        }
    }
}

function handleLimitChange() {
    const newLimit = parseInt(dom.limitSelect.value);
    state.limit = newLimit;
    state.offsetID = 0;
    state.hasMore = true;
    state.selected.clear();
    dom.grid.innerHTML = ''; // Clear grid
    updateUI();
    fetchMessages();
}



// Initial Load
fetchMessages();
