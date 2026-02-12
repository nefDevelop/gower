<script>
  import { onMount } from 'svelte';
  // Comment out real backend calls as they will fail in Linux dev mode
  // import { LoadFavorites, SetWallpaper, Blacklist, Download, RemoveFavorite, OpenURL } from '../wailsjs/go/main/App.js';

  // --- Mock Data and Functions ---
  const MOCK_FAVORITES = [
    { ID: 'fav_1', Ext: '.jpg', Permalink: '#', Thumbnail: 'https://via.placeholder.com/300x200.png?text=Favorite+1' },
    { ID: 'fav_2', Ext: '.jpg', Permalink: '#', Thumbnail: 'https://via.placeholder.com/300x200.png?text=Favorite+2' },
  ];

  let wallpapers = [];
  let loading = true;
  let error = null;

  async function loadWallpapers() {
    loading = true;
    error = null;
    console.log("Loading mock favorite wallpapers...");
    // Simulate network delay
    setTimeout(() => {
      wallpapers = MOCK_FAVORITES;
      if (wallpapers.length === 0) {
        error = 'No se encontraron wallpapers favoritos.';
      }
      loading = false;
      console.log("Mock favorites loaded:", wallpapers);
    }, 500);
  }

  // --- Mock Action Handlers ---
  function handleSet(id) { console.log(`[MOCK] Set wallpaper: ${id}`); }
  function handleBlacklist(id) {
    console.log(`[MOCK] Blacklist wallpaper: ${id}`);
    wallpapers = wallpapers.filter(w => w.ID !== id);
  }
  function handleDownload(id) { console.log(`[MOCK] Download wallpaper: ${id}`); }
  function handleRemoveFavorite(id) {
    console.log(`[MOCK] Removed favorite: ${id}`);
    wallpapers = wallpapers.filter(w => w.ID !== id);
  }
  function handleOpenPermalink(url) { console.log(`[MOCK] Open URL: ${url}`); }

  onMount(() => {
    loadWallpapers();
  });
</script>

<div>
  <h2>Favoritos (Modo Ficticio)</h2>

  {#if loading}
    <p>Cargando wallpapers favoritos...</p>
  {:else if error}
    <p class="error-message">{error}</p>
  {:else if wallpapers.length === 0 && !loading}
    <p>No tienes wallpapers favoritos.</p>
  {:else}
    <div>
      <div class="wallpaper-grid">
        {#each wallpapers as wallpaper (wallpaper.ID)}
          <div class="wallpaper-card">
             <img
                src={wallpaper.Thumbnail}
                alt={wallpaper.ID}
              />
            <div class="card-body">
              <p class="card-title">{wallpaper.ID}</p>
              <div class="card-actions">
                {#if wallpaper.Permalink}
                  <button on:click={() => handleOpenPermalink(wallpaper.Permalink)} class="link-btn">Link</button>
                {/if}
                <button
                  on:click={() => handleRemoveFavorite(wallpaper.ID)}
                  class="fav-btn is-fav"
                >
                  <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor"><path d="M11.645 20.917 3.033 12.31a.75.75 0 0 1 0-1.06L11.645 2.703a.75.75 0 0 1 1.06 0l8.612 8.614a.75.75 0 0 1 0 1.06l-8.613 8.613a.75.75 0 0 1-1.06 0Z" /></svg>
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
    </div>
  {/if}
</div>
