<script>
  import { onMount } from 'svelte';
  // Comment out real backend calls as they will fail in Linux dev mode
  // import { LoadConfig, SetConfig, ToggleDaemon, CheckDaemonStatus, GetMonitors, OpenFolderPicker, OpenFolder } from '../wailsjs/go/main/App.js';

  // --- Mock Data and Functions ---
  const MOCK_CONFIG = {
    Paths: {
      Wallpapers: "/home/user/Pictures/Wallpapers",
      IndexWallpapers: true,
    },
    Behavior: {
      ChangeInterval: 300,
      AutoDownload: true,
      RespectDarkMode: true,
      MultiMonitor: 'clone',
    },
    Power: {
      PauseOnLowBattery: true,
      LowBatteryThreshold: 20,
    },
    IsLaptop: true, // Assume it's a laptop for mock display
  };

  let config = {}; // Stores the current gower config
  let loading = true;
  let error = null;
  let daemonRunning = false;
  let monitors = [{ Name: 'eDP-1', Primary: true }]; // Mock monitor

  async function loadConfigAndStatus() {
    loading = true;
    error = null;
    console.log("Loading mock config and status...");
    // Simulate network delay
    setTimeout(() => {
      config = MOCK_CONFIG;
      daemonRunning = true; // Mock as running
      loading = false;
      console.log("Mock config loaded:", config);
    }, 500);
  }

  function handleSetConfig(key, value) {
    console.log(`[MOCK] Set config: ${key}=${value}`);
    // Optimistically update UI
    const keys = key.split('.');
    let current = config;
    for (let i = 0; i < keys.length - 1; i++) {
      if (!current[keys[i]]) current[keys[i]] = {};
      current = current[keys[i]];
    }
    current[keys[keys.length - 1]] = value;
    config = config; // Trigger Svelte reactivity
  }

  function handleToggleDaemon() {
    console.log(`[MOCK] Toggling daemon from ${daemonRunning} to ${!daemonRunning}`);
    daemonRunning = !daemonRunning; // Optimistic update
  }

  function handleOpenFolderPicker() {
    const selectedPath = "/mock/selected/path";
    console.log(`[MOCK] OpenFolderPicker would return: ${selectedPath}`);
    handleSetConfig('Paths.Wallpapers', selectedPath);
  }

  function handleOpenWallpaperFolder() {
    console.log(`[MOCK] Opening folder: ${config.Paths.Wallpapers}`);
  }

  onMount(() => {
    loadConfigAndStatus();
  });

</script>

<div class="p-4 h-full flex flex-col bg-gray-900 text-gray-100">
  <h2 class="text-2xl font-bold mb-4 text-gray-200">Ajustes (Modo Ficticio)</h2>

  {#if loading}
    <p class="text-lg text-gray-400">Cargando ajustes...</p>
  {:else if error}
    <p class="text-red-500 text-lg">{error}</p>
  {:else}
    <div class="flex-grow overflow-y-auto space-y-6">
      <!-- Daemon Status -->
      <div class="bg-gray-800 p-4 rounded-lg">
        <h3 class="text-xl font-semibold mb-2 text-gray-100">Daemon</h3>
        <p>Estado del Daemon:
          <span class="{daemonRunning ? 'text-green-500' : 'text-red-500'} font-bold">
            {daemonRunning ? 'Ejecutándose' : 'Detenido'}
          </span>
        </p>
        <button on:click={handleToggleDaemon} class="mt-2 px-4 py-2 rounded-md
          {daemonRunning ? 'bg-red-600 hover:bg-red-700' : 'bg-green-600 hover:bg-green-700'}
          text-white"
        >
          {daemonRunning ? 'Detener Daemon' : 'Iniciar Daemon'}
        </button>
      </div>

      <!-- Paths Settings -->
      <div class="bg-gray-800 p-4 rounded-lg">
        <h3 class="text-xl font-semibold mb-2 text-gray-100">Rutas</h3>
        <div class="space-y-4">
          <div>
            <label for="wallpaperPath" class="block text-sm font-medium text-gray-300">Carpeta de Wallpapers</label>
            <div class="mt-1 flex rounded-md shadow-sm">
              <input
                type="text"
                id="wallpaperPath"
                bind:value={config.Paths.Wallpapers}
                on:change={(e) => handleSetConfig('paths.wallpapers', e.target.value)}
                class="flex-grow p-2 rounded-l-md bg-gray-700 text-gray-100 border border-gray-600 focus:outline-none focus:border-blue-500"
              />
              <button on:click={handleOpenWallpaperFolder} class="px-4 py-2 bg-gray-600 text-white hover:bg-gray-500">
                Abrir
              </button>
              <button on:click={handleOpenFolderPicker} class="px-4 py-2 rounded-r-md bg-blue-600 text-white hover:bg-blue-700">
                Elegir...
              </button>
            </div>
          </div>
          <div>
            <label class="inline-flex items-center">
              <input
                type="checkbox"
                class="form-checkbox h-5 w-5 text-blue-600 bg-gray-700 border-gray-600 rounded"
                bind:checked={config.Paths.IndexWallpapers}
                on:change={(e) => handleSetConfig('paths.index_wallpapers', e.target.checked)}
              />
              <span class="ml-2 text-gray-300">Indexar Wallpapers</span>
            </label>
          </div>
        </div>
      </div>

      <!-- Behavior Settings -->
      <div class="bg-gray-800 p-4 rounded-lg">
        <h3 class="text-xl font-semibold mb-2 text-gray-100">Comportamiento</h3>
        <div class="space-y-4">
          <div>
            <label for="interval" class="block text-sm font-medium text-gray-300">Intervalo de cambio (segundos)</label>
            <input
              type="number"
              id="interval"
              bind:value={config.Behavior.ChangeInterval}
              on:change={(e) => handleSetConfig('behavior.change_interval', Number(e.target.value))}
              class="mt-1 p-2 block w-full rounded-md bg-gray-700 text-gray-100 border border-gray-600 focus:outline-none focus:border-blue-500"
            />
          </div>
          <div>
            <label class="inline-flex items-center">
              <input
                type="checkbox"
                class="form-checkbox h-5 w-5 text-blue-600 bg-gray-700 border-gray-600 rounded"
                bind:checked={config.Behavior.AutoDownload}
                on:change={(e) => handleSetConfig('behavior.auto_download', e.target.checked)}
              />
              <span class="ml-2 text-gray-300">Descarga automática</span>
            </label>
          </div>
          <div>
            <label class="inline-flex items-center">
              <input
                type="checkbox"
                class="form-checkbox h-5 w-5 text-blue-600 bg-gray-700 border-gray-600 rounded"
                bind:checked={config.Behavior.RespectDarkMode}
                on:change={(e) => handleSetConfig('behavior.respect_dark_mode', e.target.checked)}
              />
              <span class="ml-2 text-gray-300">Respetar modo oscuro</span>
            </label>
          </div>
          <div>
            <label for="multiMonitor" class="block text-sm font-medium text-gray-300">Modo multi-monitor</label>
            <select
              id="multiMonitor"
              bind:value={config.Behavior.MultiMonitor}
              on:change={(e) => handleSetConfig('behavior.multi_monitor', e.target.value)}
              class="mt-1 p-2 block w-full rounded-md bg-gray-700 text-gray-100 border border-gray-600 focus:outline-none focus:border-blue-500"
            >
              <option value="clone">Clonar</option>
              <option value="distinct">Distinto</option>
            </select>
          </div>
        </div>
      </div>

      <!-- Power Settings -->
      <div class="bg-gray-800 p-4 rounded-lg">
        <h3 class="text-xl font-semibold mb-2 text-gray-100">Energía</h3>
        <div class="space-y-4">
          {#if config.IsLaptop}
          <div>
            <label class="inline-flex items-center">
              <input
                type="checkbox"
                class="form-checkbox h-5 w-5 text-blue-600 bg-gray-700 border-gray-600 rounded"
                bind:checked={config.Power.PauseOnLowBattery}
                on:change={(e) => handleSetConfig('power.pause_on_low_battery', e.target.checked)}
              />
              <span class="ml-2 text-gray-300">Pausar con batería baja</span>
            </label>
          </div>
          <div>
            <label for="batteryThreshold" class="block text-sm font-medium text-gray-300">Umbral de batería baja (%)</label>
            <input
              type="number"
              id="batteryThreshold"
              bind:value={config.Power.LowBatteryThreshold}
              on:change={(e) => handleSetConfig('power.low_battery_threshold', Number(e.target.value))}
              class="mt-1 p-2 block w-full rounded-md bg-gray-700 text-gray-100 border border-gray-600 focus:outline-none focus:border-blue-500"
            />
          </div>
          {:else}
            <p class="text-gray-400">Ajustes de energía solo disponibles en portátiles.</p>
          {/if}
        </div>
      </div>

      <!-- Providers Settings (simplified) -->
      <div class="bg-gray-800 p-4 rounded-lg">
        <h3 class="text-xl font-semibold mb-2 text-gray-100">Proveedores</h3>
        <p class="text-gray-400">Gestión de proveedores avanzada se implementará más adelante.</p>
        <!-- Dynamic list of providers can be added here -->
      </div>
    </div>
  {/if}
</div>

<style>
  /* You can add specific styles for SettingsTab here if needed */
</style>
