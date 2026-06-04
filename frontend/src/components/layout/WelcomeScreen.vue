<template>
  <div class="welcome">
    <div class="welcome-content">
      <div class="welcome-logo">
        <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
          <rect x="3" y="3" width="18" height="18" rx="2"/>
          <line x1="3" y1="9" x2="21" y2="9"/>
          <line x1="3" y1="15" x2="21" y2="15"/>
          <line x1="9" y1="3" x2="9" y2="21"/>
        </svg>
      </div>
      <h1 class="welcome-title">OpenCSV</h1>
      <p class="welcome-subtitle">The open-source CSV editor for modern data workflows</p>

      <div class="welcome-actions">
        <button class="btn btn-primary welcome-btn" @click="openFile">
          <FolderOpen :size="16" />
          Open CSV File
        </button>
        <input ref="fileInput" type="file" accept=".csv,.tsv,.txt,.xlsx" style="display:none" @change="onFileInput" multiple />
      </div>

      <div class="welcome-features">
        <div class="feature-item">
          <Zap :size="14" />
          <span>100MB+ files open in seconds</span>
        </div>
        <div class="feature-item">
          <Database :size="14" />
          <span>SQL queries on CSV data</span>
        </div>
        <div class="feature-item">
          <ArrowUpDown :size="14" />
          <span>Sort, filter, find & replace</span>
        </div>
        <div class="feature-item">
          <FileSpreadsheet :size="14" />
          <span>Excel import & export</span>
        </div>
      </div>

      <div class="welcome-shortcuts">
        <span class="kbd">⌘O</span> Open &nbsp;
        <span class="kbd">⌘S</span> Save &nbsp;
        <span class="kbd">⌘P</span> Command palette &nbsp;
        <span class="kbd">⌘F</span> Find & Replace
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { FolderOpen, Zap, Database, ArrowUpDown, FileSpreadsheet } from 'lucide-vue-next'
import { useFileOpener } from '@/composables/useFileOpener'

const { fileInput, openFile, onFileInput } = useFileOpener()
</script>

<style scoped>
.welcome {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--bg-app);
}
.welcome-content {
  text-align: center;
  max-width: 480px;
  padding: 40px;
}
.welcome-logo {
  color: var(--accent);
  margin-bottom: 16px;
  display: flex;
  justify-content: center;
  opacity: 0.8;
}
.welcome-title {
  font-family: var(--font-mono);
  font-size: 28px;
  font-weight: 600;
  letter-spacing: -1px;
  color: var(--text-primary);
  margin-bottom: 8px;
}
.welcome-subtitle {
  font-size: 14px;
  color: var(--text-secondary);
  margin-bottom: 32px;
  line-height: 1.6;
}
.welcome-actions { margin-bottom: 32px; }
.welcome-btn {
  font-size: 14px;
  padding: 10px 24px;
  gap: 8px;
  cursor: pointer;
}
.welcome-features {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 10px;
  margin-bottom: 28px;
  text-align: left;
}
.feature-item {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  color: var(--text-secondary);
  padding: 8px 12px;
  background: var(--bg-surface);
  border: 1px solid var(--border);
  border-radius: var(--radius);
}
.feature-item svg { color: var(--accent); flex-shrink: 0; }
.welcome-shortcuts {
  font-size: 11px;
  color: var(--text-muted);
}
.kbd {
  display: inline-block;
  padding: 1px 5px;
  background: var(--bg-surface-3);
  border: 1px solid var(--border-strong);
  border-radius: 3px;
  font-family: var(--font-mono);
  font-size: 10px;
}
</style>
