const resizePane = function(map) {
    const divider = document.querySelector('.divider');
    const leftPanel = document.querySelector('.left-panel');
    const rightPanel = document.querySelector('.right-panel');

    let isResizing = false;

    divider.addEventListener('mousedown', function(e) {
        isResizing = true;
    });

    document.addEventListener('mousemove', function(e) {
        if (!isResizing) return;

        // Calculate new width for the left panel
        const newLeftWidth = e.clientX / window.innerWidth * 100;

        if (newLeftWidth < 10 || newLeftWidth > 90) return;  // Limit resizing

        leftPanel.style.width = `${newLeftWidth}%`;
        rightPanel.style.width = `${100 - newLeftWidth}%`;

        map.resize();
    });

    document.addEventListener('mouseup', function(e) {
        isResizing = false;
    });
}

export { resizePane };
