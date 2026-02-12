<script>
  import { onMount } from 'svelte';
  // Import tab components
  import HomeTab from './HomeTab.svelte';
  import SearchTab from './SearchTab.svelte';
  import FavoritesTab from './FavoritesTab.svelte';
  import SettingsTab from './SettingsTab.svelte';

  let currentTab = 'home'; // State to manage active tab

  const tabs = [
    { id: 'home', label: 'Inicio', component: HomeTab },
    { id: 'search', label: 'Buscar', component: SearchTab },
    { id: 'favorites', label: 'Favoritos', component: FavoritesTab },
    { id: 'settings', label: 'Ajustes', component: SettingsTab },
  ];

  // Placeholder for any initial app setup or data loading
  onMount(() => {
    // Example: call Go backend Initialize method
    // window.go.main.App.Initialize().then(() => {
    //   console.log("App initialized from Go backend");
    // }).catch(err => {
    //   console.error("Failed to initialize app:", err);
    // });
  });
</script>

<main class="app-container">
  <!-- Tab Navigation Bar -->
  <nav class="tab-nav">
    {#each tabs as tab}
      <button
        class:active={currentTab === tab.id}
        on:click={() => (currentTab = tab.id)}
      >
        {tab.label}
      </button>
    {/each}
  </nav>

  <!-- Tab Content Area -->
  <div class="tab-content">
    {#each tabs as tab}
      {#if currentTab === tab.id}
        <svelte:component this={tab.component} />
      {/if}
    {/each}
  </div>
</main>
