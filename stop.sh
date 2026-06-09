#!/bin/bash
# =============================================================================
# Stop submill and mihomo
# =============================================================================
echo ">>> Stopping services..."

if pkill -f "/mihomo/mihomo" 2>/dev/null; then
    echo "    mihomo stopped"
else
    echo "    mihomo not running"
fi

if pkill -f "^/.*/submill" 2>/dev/null; then
    echo "    submill stopped"
else
    echo "    submill not running"
fi

echo ">>> Done"