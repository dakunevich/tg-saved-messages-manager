const state = {
    offsetID: 0,
    isLoading: false,
    selected: new Set(),
    hasMore: true,
    limit: 20
};

const dom = {
    grid: document.getElementById('message-grid'),
    loadMoreBtn: document.getElementById('load-more-btn'),
    deleteBtn: document.getElementById('delete-btn'),
    selectionCount: document.getElementById('selection-count'),
    loader: document.getElementById('loader'),
    totalCount: document.getElementById('total-count'),
    limitSelect: document.getElementById('limit-select'),
    selectEmptyBtn: document.getElementById('select-empty-btn')
};

console.log('[DEBUG] DOM Elements:', dom);

function logAction(message) {
    console.log(`[UI LOG] ${message}`);
}

async function fetchMessages() {
    if (state.isLoading || !state.hasMore) return;

    state.isLoading = true;
    dom.loadMoreBtn.disabled = true;
    dom.loadMoreBtn.textContent = "Loading...";

    try {
        logAction(`Fetching messages (limit: ${state.limit}, offset: ${state.offsetID})...`);
        const res = await fetch(`/api/messages?limit=${state.limit}&offset_id=${state.offsetID}&_t=${Date.now()}`);
        if (!res.ok) throw new Error('Failed to fetch');

        const data = await res.json();
        console.log('[DEBUG] Received data:', data);

        // Update Total Count
        if (data.total !== undefined) {
            if (dom.totalCount) dom.totalCount.textContent = data.total;
        } else {
            console.warn('[DEBUG] Total count missing in response');
        }

        if (!data.messages || data.messages.length === 0) {
            state.hasMore = false;
            dom.loadMoreBtn.style.display = 'none';
            logAction("No more messages found.");
            return;
        }

        renderMessages(data.messages);
        logAction(`Loaded ${data.messages.length} messages.`);

        // Update offset for next page
        const lastMsg = data.messages[data.messages.length - 1];
        state.offsetID = lastMsg.id;

    } catch (err) {
        console.error(err);
        alert('Error loading messages');
    } finally {
        state.isLoading = false;
        dom.loadMoreBtn.disabled = false;
        dom.loadMoreBtn.textContent = "Load More";
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
            contentHtml = msg.message;
        } else {
            contentHtml = '<i>(No text content)</i>';
            card.classList.add('is-empty');
        }

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
            // Fallback for old/single style if any
            if (msg.media_type === "WebLink" && msg.web_preview) {
                // Do not render "WebLink" tag if we have a preview
            } else {
                if (msg.media_type === "Photo") {
                    mediaHtml = `<div style="margin-bottom: 8px;"><img src="/api/media?id=${msg.id}" loading="lazy" alt="Photo ${msg.id}"></div>`;
                } else {
                    mediaHtml = `<span class="media-tag" style="margin-bottom: 8px; display:inline-block;">${msg.media_type}</span>`;
                }
            }
        }

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

        // We store the list of IDs in the checkbox value as JSON string for easy retrieval
        // Or we use data attribute.
        // Let's use data attribute on card for access.
        card.dataset.ids = JSON.stringify(msg.ids || [msg.id]);

        card.innerHTML = `
            <div class="checkbox-wrapper">
                <input type="checkbox" value="${msg.id}"> <!-- Value is just main ID for reference, invalid for logic now -->
            </div>
            <div class="meta">
                <span>ID: ${msg.id} ${msg.ids && msg.ids.length > 1 ? `(+${msg.ids.length - 1})` : ''}</span>
                <span>${dateStr}</span>
            </div>
            ${mediaHtml}
            <div class="content">
                ${contentHtml}
            </div>
            ${previewHtml}
        `;

        // Event Listeners
        const checkbox = card.querySelector('input');
        // We pass the full list of IDs to toggleSelection
        const allIds = msg.ids || [msg.id];

        checkbox.addEventListener('change', (e) => toggleSelection(allIds, e.target.checked));

        card.addEventListener('click', (e) => {
            if (e.target !== checkbox && e.target.tagName !== 'IMG') {
                checkbox.checked = !checkbox.checked;
                toggleSelection(allIds, checkbox.checked);
            }
        });

        dom.grid.appendChild(card);
    });
}

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
        // We iterate state.selected or ids.
        // But cards have many IDs. If we delete 1 ID from a group, what happens?
        // Usually we delete all.
        // We need to remove the card corresponding to the standard ID.
        // We can just iterate all cards and see if their IDs are in the deleted set.

        document.querySelectorAll('.message-card').forEach(card => {
            const cardIds = JSON.parse(card.dataset.ids || "[]");
            // If any of cardIds is deleted, remove card?
            // Or if ALL are deleted?
            // Since we group select, we likely delete all.
            // Let's remove card if its main ID is deleted.
            const mainId = parseInt(card.dataset.id);
            if (ids.includes(mainId)) {
                card.remove();
            }
        });

        state.selected.clear();
        updateUI();
        logAction(`Successfully deleted ${ids.length} messages.`);

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

dom.loadMoreBtn.addEventListener('click', fetchMessages);
dom.deleteBtn.addEventListener('click', deleteSelected);
dom.selectEmptyBtn.addEventListener('click', selectEmpty);
dom.limitSelect.addEventListener('change', handleLimitChange);

// Initial Load
fetchMessages();
