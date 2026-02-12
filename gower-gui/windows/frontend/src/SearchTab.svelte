<script>
  import { onMount } from 'svelte';
  // Comment out real backend calls as they will fail in Linux dev mode
  // import { Search, SetWallpaper, Blacklist, Download, AddFavorite, RemoveFavorite, LoadFavorites, OpenURL, LoadConfig } from '../wailsjs/go/main/App.js';

  // --- Mock Data and Functions ---
  const MOCK_PROVIDERS = [
    { key: 'wallhaven', name: 'Wallhaven' },
    { key: 'unsplash', name: 'Unsplash' },
  ];
  const MOCK_SEARCH_RESULTS = [
    { ID: 'search_1', Ext: '.jpg', Permalink: '#', Thumbnail: 'https://via.placeholder.com/300x200.png?text=Search+Result+1' },
    { ID: 'search_2', Ext: '.jpg', Permalink: '#', Thumbnail: 'https://via.placeholder.com/300x200.png?text=Search+Result+2' },
  ];
  const MOCK_FAVORITES = ['search_2'];


  let searchQuery = '';
  let selectedProvider = '';
  let providers = []; // List of available providers
  let wallpapers = [];
  let loading = false;
  let error = null;
  let favoritesList = []; // List of favorite wallpaper IDs

  async function loadInitialData() {
    console.log("Loading mock providers and favorites...");
    providers = MOCK_PROVIDERS;
    if (providers.length > 0) {
      selectedProvider = providers[0].key;
    }
    favoritesList = MOCK_FAVORITES;
  }

  async function handleSearch() {
    loading = true;
    error = null;
    wallpapers = []; // Clear previous results

    if (!searchQuery) {
      error = 'Please enter a search query.';
      loading = false;
      return;
    }
    if (!selectedProvider) {
      error = 'Please select a provider.';
      loading = false;
      return;
    }

    console.log(`[MOCK] Searching for "${searchQuery}" on provider "${selectedProvider}"...`);
    // Simulate network delay
    setTimeout(() => {
      // Return slightly different results based on query to make it feel more real
      if (searchQuery.toLowerCase().includes('cat')) {
        wallpapers = MOCK_SEARCH_RESULTS.map(w => ({...w, Thumbnail: w.Thumbnail.replace('Result', 'Cat')}));
      } else {
        wallpapers = MOCK_SEARCH_RESULTS;
      }
      loading = false;
    }, 1000);
  }

  // --- Mock Action Handlers (similar to HomeTab) ---
  function handleSet(id) { console.log(`[MOCK] Set wallpaper: ${id}`); }
  function handleBlacklist(id) {
    console.log(`[MOCK] Blacklist wallpaper: ${id}`);
    wallpapers = wallpapers.filter(w => w.ID !== id);
  }
  function handleDownload(id) { console.log(`[MOCK] Download wallpaper: ${id}`); }
  function handleToggleFavorite(wallpaper) {
    const isFav = favoritesList.includes(wallpaper.ID);
    if (isFav) {
      favoritesList = favoritesList.filter(id => id !== wallpaper.ID);
      console.log(`[MOCK] Removed favorite: ${wallpaper.ID}`);
    } else {
      favoritesList = [...favoritesList, wallpaper.ID];
      console.log(`[MOCK] Added favorite: ${wallpaper.ID}`);
    }
  }
  function handleOpenPermalink(url) { console.log(`[MOCK] Open URL: ${url}`); }

  onMount(() => {
    loadInitialData();
  });
</script>

<div class="p-4 h-full flex flex-col bg-gray-900 text-gray-100">
  <h2 class="text-2xl font-bold mb-4 text-gray-200">Buscar Wallpapers (Modo Ficticio)</h2>

  <div class="mb-4 flex space-x-4">
    <input
      type="text"
      bind:value={searchQuery}
      placeholder="Introduce tu búsqueda..."
      class="flex-grow p-2 rounded-md bg-gray-700 text-gray-100 border border-gray-600 focus:outline-none focus:border-blue-500"
      on:keydown={(e) => { if (e.key === 'Enter') handleSearch(); }}
    />
    <select
      bind:value={selectedProvider}
      class="p-2 rounded-md bg-gray-700 text-gray-100 border border-gray-600 focus:outline-none focus:border-blue-500"
    >
      {#each providers as provider (provider.key)}
        <option value={provider.key}>{provider.name}</option>
      {/each}
    </select>
    <button on:click={handleSearch} class="bg-blue-500 text-white px-4 py-2 rounded-md hover:bg-blue-600">Buscar</button>
  </div>

  {#if loading}
    <p class="text-lg text-gray-400">Buscando wallpapers...</p>
  {:else if error}
    <p class="text-red-500 text-lg">{error}</p>
  {:else if wallpapers.length === 0 && !loading}
    <!-- Don't show a message initially, wait for a search to be performed -->
  {:else}
    <div class="flex-grow overflow-y-auto">
      <div class="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-4">
        {#each wallpapers as wallpaper (wallpaper.ID)}
          <div class="bg-gray-800 rounded-lg shadow-md overflow-hidden relative group">
             <img
                src={wallpaper.Thumbnail}
                alt={wallpaper.ID}
                class="w-full h-48 object-cover transition-transform duration-300 group-hover:scale-105"
              />
            <div class="p-3">
              <p class="text-sm font-semibold text-gray-300 truncate">{wallpaper.ID}</p>
              <div class="flex items-center gap-1 mt-1">
                {#if wallpaper.Permalink}
                  <button on:click={() => handleOpenPermalink(wallpaper.Permalink)} class="text-xs text-blue-400 hover:underline">Link</button>
                {/if}
                <button
                  on:click={() => handleToggleFavorite(wallpaper)}
                  class="ml-auto text-sm p-1 rounded-full
                         {favoritesList.includes(wallpaper.ID) ? 'text-yellow-400' : 'text-gray-400 hover:text-yellow-300'}"
                >
                  {#if favoritesList.includes(wallpaper.ID)}
                    <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor" class="w-5 h-5"><path fill-rule="evenodd" d="M10.788 3.21c.448-1.077 1.976-1.077 2.424 0l2.082 5.006 5.404.434c1.164.093 1.636 1.545.749 2.305l-4.117 3.527 1.257 5.273c.271 1.136-.964 2.033-1.96 1.425L12 18.354l-4.697 2.81c-.996.608-2.231-.29-1.96-1.425l1.257-5.273-4.117-3.527c-.887-.76-.415-2.212.749-2.305l5.404-.434 2.082-5.005Z" clip-rule="evenodd" /></svg>
                  {:else}
                    <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-5 h-5"><path stroke-linecap="round" stroke-linejoin="round" d="M11.48 3.499a.562.562 0 0 1 1.04 0l2.123 5.116 5.418.433a.562.562 0 0 1 .314.997l-4.117 3.527 1.257 5.273c.117.491-.464.935-.837.641l-4.75-2.85-4.75 2.85c-.373.294-.954-.15-.837-.641l1.257-5.273-4.117-3.527a.562.562 0 0 1 .314-.997l5.418-.433L11.48 3.5Z" /></svg>
                  {/if}
                </button>
              </div>
            </div>
            <div class="absolute inset-0 bg-black bg-opacity-50 flex items-center justify-center space-x-2 opacity-0 group-hover:opacity-100 transition-opacity duration-300">
                <button on:click={() => handleSet(wallpaper.ID)} class="bg-blue-500 text-white px-3 py-1 rounded-md text-sm hover:bg-blue-600">Set</button>
                <button on:click={() => handleBlacklist(wallpaper.ID)} class="bg-red-500 text-white px-3 py-1 rounded-md text-sm hover:bg-red-600">Blacklist</button>
                <button on:click={() => handleDownload(wallpaper.ID)} class="bg-green-500 text-white px-3 py-1 rounded-md text-sm hover:bg-green-600">Download</button>
            </div>
          </div>
        {/each}
      </div>
    </div>
  {/if}
</div>

<style>
  /* You can add specific styles for SearchTab here if needed */
</style>
