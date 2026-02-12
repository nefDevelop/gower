<script>
  import { onMount } from 'svelte';
  // Comment out real backend calls as they will fail in Linux dev mode
  // import { SetWallpaper, Blacklist, Download, AddFavorite, RemoveFavorite, LoadFavorites, OpenURL, LoadFeed } from '../wailsjs/go/main/App.js';

  // --- Mock Data and Functions ---
  const MOCK_WALLPAPERS = [
    { ID: 'mock_1', Ext: '.jpg', Permalink: '#', Thumbnail: 'https://via.placeholder.com/300x200.png?text=Mock+1' },
    { ID: 'mock_2', Ext: '.jpg', Permalink: '#', Thumbnail: 'https://via.placeholder.com/300x200.png?text=Mock+2' },
    { ID: 'mock_3', Ext: '.jpg', Permalink: '#', Thumbnail: 'https://via.placeholder.com/300x200.png?text=Mock+3' },
    { ID: 'mock_4', Ext: '.jpg', Permalink: '#', Thumbnail: 'https://via.placeholder.com/300x200.png?text=Mock+4' },
    { ID: 'mock_5', Ext: '.jpg', Permalink: '#', Thumbnail: 'https://via.placeholder.com/300x200.png?text=Mock+5' },
    { ID: 'mock_6', Ext: '.jpg', Permalink: '#', Thumbnail: 'https://via.placeholder.com/300x200.png?text=Mock+6' },
  ];

  let wallpapers = [];
  let loading = true;
  let error = null;
  let page = 1;
  let hasMore = true;
  let favoritesList = ['mock_2', 'mock_5']; // Mock some favorites

  async function loadWallpapers() {
    loading = true;
    error = null;
    console.log("Loading mock wallpapers...");
    // Simulate network delay
    setTimeout(() => {
      if (page > 2) { // Simulate running out of wallpapers
          hasMore = false;
          loading = false;
          return;
      }
      const newWallpapers = MOCK_WALLPAPERS.map(w => ({...w, ID: `${w.ID}_page${page}`}));
      wallpapers = [...wallpapers, ...newWallpapers];
      page++;
      loading = false;
      console.log("Mock wallpapers loaded:", wallpapers);
    }, 1000);
  }

  // --- Mock Action Handlers ---
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
    // We don't need to load real favorites, just the mock wallpapers
    loadWallpapers();
  });
</script>

<div>
  <h2>Inicio (Modo Ficticio)</h2>

  {#if loading && wallpapers.length === 0}
    <p>Cargando wallpapers...</p>
  {:else if error}
    <p class="error-message">{error}</p>
  {:else if wallpapers.length === 0 && !loading}
    <p>No se encontraron wallpapers.</p>
  {:else}
    <div>
      <div class="wallpaper-grid">
        {#each wallpapers as wallpaper (wallpaper.ID)}
          <div class="wallpaper-card">
            {#if wallpaper.Thumbnail}
              <img
                src={wallpaper.Thumbnail}
                alt={wallpaper.ID}
              />
            {:else}
              <div class="no-preview">
                No Preview
              </div>
            {/if}
            <div class="card-body">
              <p class="card-title">{wallpaper.ID}</p>
              <div class="card-actions">
                 {#if wallpaper.Permalink}
                  <button on:click={() => handleOpenPermalink(wallpaper.Permalink)} class="link-btn">Link</button>
                {/if}
                 <button
                  on:click={() => handleToggleFavorite(wallpaper)}
                  class="fav-btn"
                  class:is-fav={favoritesList.includes(wallpaper.ID)}
                >
                  {#if favoritesList.includes(wallpaper.ID)}
                    <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor"><path fill-rule="evenodd" d="M10.788 3.21c.448-1.077 1.976-1.077 2.424 0l2.082 5.006 5.404.434c1.164.093 1.636 1.545.749 2.305l-4.117 3.527 1.257 5.273c.271 1.136-.964 2.033-1.96 1.425L12 18.354l-4.697 2.81c-.996.608-2.231-.29-1.96-1.425l1.257-5.273-4.117-3.527c-.887-.76-.415-2.212.749-2.305l5.404-.434 2.082-5.005Z" clip-rule="evenodd" /></svg>
                  {:else}
                    <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" d="M11.48 3.499a.562.562 0 0 1 1.04 0l2.123 5.116 5.418.433a.562.562 0 0 1 .314.997l-4.117 3.527 1.257 5.273c.117.491-.464.935-.837.641l-4.75-2.85-4.75 2.85c-.373.294-.954-.15-.837-.641l1.257-5.273-4.117-3.527a.562.562 0 0 1 .314-.997l5.418-.433L11.48 3.5Z" /></svg>
                  {/if}
                </button>
              </div>
            </div>
            <div class="overlay">
                <button on:click={() => handleSet(wallpaper.ID)}>Set</button>
                <button on:click={() => handleBlacklist(wallpaper.ID)} class="blacklist-btn">Blacklist</button>
                <button on:click={() => handleDownload(wallpaper.ID)} class="download-btn">Download</button>
            </div>
          </div>
        {/each}
      </div>

      {#if loading && wallpapers.length > 0}
        <p>Cargando más...</p>
      {:else if hasMore}
        <div style="text-align: center; margin-top: 1rem;">
          <button on:click={() => loadWallpapers()} class="primary-btn">
            Cargar más
          </button>
        </div>
      {/if}
    </div>
  {/if}
</div>
