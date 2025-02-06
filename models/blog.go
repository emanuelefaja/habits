package models

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"gopkg.in/yaml.v3"
)

// BlogPost represents a single blog post
type BlogPost struct {
	Title       string        `yaml:"title"`
	Slug        string        `yaml:"slug"`
	Date        time.Time     `yaml:"date"`
	Emoji       string        `yaml:"emoji"`
	Description string        `yaml:"description"` // Meta description for SEO and previews
	Tags        []string      `yaml:"tags"`
	Content     string        // Raw markdown content
	HTML        template.HTML // Rendered HTML content
}

// BlogService handles all blog-related operations
type BlogService struct {
	posts    []*BlogPost
	postsMap map[string]*BlogPost // Slug to post mapping for quick lookups
	mu       sync.RWMutex         // Protects posts and postsMap
}

var (
	blogService *BlogService
	once        sync.Once
)

// GetBlogService returns a singleton instance of BlogService
func GetBlogService() *BlogService {
	once.Do(func() {
		blogService = &BlogService{
			postsMap: make(map[string]*BlogPost),
		}
		err := blogService.LoadPosts()
		if err != nil {
			// In production, you might want to handle this differently
			panic(err)
		}
	})
	return blogService
}

// LoadPosts reads all markdown files from the blog/posts directory
// and parses them into BlogPost structs
func (bs *BlogService) LoadPosts() error {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	// Clear existing posts
	bs.posts = nil
	bs.postsMap = make(map[string]*BlogPost)

	// Create markdown parser with extensions
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,           // GitHub Flavored Markdown
			extension.Table,         // Tables support
			extension.Strikethrough, // Strikethrough support
		),
	)

	// Walk through all .md files in the blog directory
	postsDir := "content/blog"
	err := filepath.Walk(postsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip if not a markdown file
		if info.IsDir() || filepath.Ext(path) != ".md" {
			return nil
		}

		// Read the file
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		// Split front matter and content
		post, err := bs.parsePost(content, md)
		if err != nil {
			return err
		}

		// Add to our collections
		bs.posts = append(bs.posts, post)
		bs.postsMap[post.Slug] = post

		return nil
	})

	if err != nil {
		return err
	}

	// Sort posts by date (newest first)
	sort.Slice(bs.posts, func(i, j int) bool {
		return bs.posts[i].Date.After(bs.posts[j].Date)
	})

	return nil
}

// parsePost splits and parses the YAML front matter and markdown content
func (bs *BlogService) parsePost(content []byte, md goldmark.Markdown) (*BlogPost, error) {
	// Split front matter and content
	parts := bytes.Split(content, []byte("---\n"))
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid post format: no front matter found")
	}

	// Parse front matter
	post := &BlogPost{}
	err := yaml.Unmarshal(parts[1], post)
	if err != nil {
		return nil, fmt.Errorf("error parsing front matter: %v", err)
	}

	// Join the remaining parts as content (in case there are more --- separators in the content)
	markdownContent := bytes.Join(parts[2:], []byte("---\n"))
	post.Content = string(markdownContent)

	// Convert markdown to HTML
	var buf bytes.Buffer
	if err := md.Convert(markdownContent, &buf); err != nil {
		return nil, fmt.Errorf("error converting markdown: %v", err)
	}
	post.HTML = template.HTML(buf.String())

	return post, nil
}

// GetPost retrieves a single post by its slug
func (bs *BlogService) GetPost(slug string) (*BlogPost, bool) {
	bs.mu.RLock()
	defer bs.mu.RUnlock()

	post, exists := bs.postsMap[slug]
	return post, exists
}

// GetAllPosts returns all posts sorted by date (newest first)
func (bs *BlogService) GetAllPosts() []*BlogPost {
	bs.mu.RLock()
	defer bs.mu.RUnlock()

	// Return a copy to prevent modification of the internal slice
	posts := make([]*BlogPost, len(bs.posts))
	copy(posts, bs.posts)
	return posts
}
