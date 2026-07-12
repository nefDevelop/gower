package core

import (
	"os"
	"path/filepath"
	"time"

	"gower/internal/utils"
	"gower/pkg/models"
)

type FeedService struct {
	Config      *models.Config
	feedManager *utils.SecureJSONManager
	Log         *utils.Logger
}

func (s *FeedService) log() *utils.Logger {
	if s.Log != nil {
		return s.Log
	}
	return utils.Log
}

func NewFeedService(cfg *models.Config, fm *utils.SecureJSONManager) *FeedService {
	return &FeedService{
		Config:      cfg,
		feedManager: fm,
	}
}

func (s *FeedService) getFeedPath() (string, error) {
	appDir, err := GetAppDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(appDir, "data", "feed.json"), nil
}

func (s *FeedService) getFeedCachePath() (string, error) {
	appDir, err := GetAppDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(appDir, "data", "feed_cache.json"), nil
}

func (s *FeedService) getBlacklistPath() (string, error) {
	appDir, err := GetAppDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(appDir, "data", "blacklist.json"), nil
}

func (s *FeedService) getFavoritesPath() string {
	appDir, err := GetAppDir()
	if err != nil {
		return ""
	}
	return filepath.Join(appDir, "data", "favorites.json")
}

func (s *FeedService) getFeedPathString() string {
	appDir, err := GetAppDir()
	if err != nil {
		return ""
	}
	return filepath.Join(appDir, "data", "feed.json")
}

func (s *FeedService) loadBlacklist() ([]string, error) {
	path, err := s.getBlacklistPath()
	if err != nil {
		return nil, err
	}
	var blacklist []string
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return []string{}, nil
	}
	if err := s.feedManager.ReadJSON(path, &blacklist); err != nil {
		return nil, err
	}
	return blacklist, nil
}

func (s *FeedService) saveBlacklist(blacklist []string) error {
	path, err := s.getBlacklistPath()
	if err != nil {
		return err
	}
	return s.feedManager.WriteJSON(path, blacklist)
}

func (s *FeedService) loadFeed() ([]models.Wallpaper, error) {
	path, err := s.getFeedPath()
	if err != nil {
		return nil, err
	}
	var feed []models.Wallpaper
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return []models.Wallpaper{}, nil
	}
	if err := s.feedManager.ReadJSON(path, &feed); err != nil {
		return nil, err
	}
	return feed, nil
}

func (s *FeedService) saveFeed(feed []models.Wallpaper) error {
	path, err := s.getFeedPath()
	if err != nil {
		return err
	}
	return s.feedManager.WriteJSON(path, feed)
}

func (s *FeedService) loadFavorites() ([]FavoriteWallpaper, error) {
	path := s.getFavoritesPath()
	if path == "" {
		return nil, nil
	}
	var favorites []FavoriteWallpaper
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return []FavoriteWallpaper{}, nil
	}
	if err := s.feedManager.ReadJSON(path, &favorites); err != nil {
		return nil, err
	}
	return favorites, nil
}

func (s *FeedService) saveFavorites(favorites []FavoriteWallpaper) error {
	path := s.getFavoritesPath()
	if path == "" {
		return nil
	}
	return s.feedManager.WriteJSON(path, favorites)
}

func (s *FeedService) loadFeedCache() ([]FeedCache, error) {
	path, err := s.getFeedCachePath()
	if err != nil {
		return nil, err
	}
	var caches []FeedCache
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return []FeedCache{}, nil
	}
	if err := s.feedManager.ReadJSON(path, &caches); err != nil {
		return nil, err
	}
	return caches, nil
}

func (s *FeedService) saveFeedCache(caches []FeedCache) error {
	path, err := s.getFeedCachePath()
	if err != nil {
		return err
	}
	return s.feedManager.WriteJSON(path, caches)
}

func (s *FeedService) SaveParserSearch(providerName, query string, results []models.Wallpaper) error {
	appDir, err := GetAppDir()
	if err != nil {
		return err
	}
	path := filepath.Join(appDir, "data", "parser", providerName+".json")

	var searches []ParserSearch
	if _, err := os.Stat(path); err == nil {
		_ = s.feedManager.ReadJSON(path, &searches)
	}

	cutoff := time.Now().AddDate(0, 0, -14)
	var validSearches []ParserSearch
	for _, s := range searches {
		if s.Date.After(cutoff) {
			validSearches = append(validSearches, s)
		}
	}

	newSearch := ParserSearch{
		Date:    time.Now(),
		Query:   query,
		Results: results,
	}
	validSearches = append(validSearches, newSearch)

	return s.feedManager.WriteJSON(path, validSearches)
}

func (s *FeedService) LoadParserSearch(providerName, query string) ([]models.Wallpaper, bool) {
	appDir, err := GetAppDir()
	if err != nil {
		return nil, false
	}
	path := filepath.Join(appDir, "data", "parser", providerName+".json")

	var searches []ParserSearch
	if _, err := os.Stat(path); err == nil {
		if err := s.feedManager.ReadJSON(path, &searches); err != nil {
			return nil, false
		}
	}

	for _, search := range searches {
		if search.Query == query {
			return search.Results, true
		}
	}
	return nil, false
}
