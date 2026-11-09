package search

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/elastic/go-elasticsearch/v9"
	"github.com/elastic/go-elasticsearch/v9/typedapi/types"
	"github.com/elastic/go-elasticsearch/v9/typedapi/types/enums/refresh"
	"github.com/elastic/go-elasticsearch/v9/typedapi/types/enums/result"
	"github.com/elastic/go-elasticsearch/v9/typedapi/types/enums/sortorder"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/errgroup"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"

	"github.com/Lil-Strudel/2026-cn-hackathon/internal/captions"
)

const (
	// Leaves room under captions.ValidTalkID's 64 characters for a "-100" suffix.
	maxSlugLength = 48
	maxSlugTries  = 100
	maxTalks      = 1000
	maxSearchHits = 50
	// Enough to show a talk keeps coming back to a term without flooding the results.
	transcriptHitsPerTalk = 3
)

type TalkStatus string

const (
	TalkLive     TalkStatus = "live"
	TalkFinished TalkStatus = "finished"
)

// Talk is the web app's Talk.
type Talk struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Speaker     string     `json:"speaker"`
	Description string     `json:"description"`
	Status      TalkStatus `json:"status"`
	StartedAt   time.Time  `json:"startedAt"`
	EndedAt     *time.Time `json:"endedAt"`
}

type NewTalk struct {
	Title       string `json:"title"`
	Speaker     string `json:"speaker"`
	Description string `json:"description"`
}

type Highlight struct {
	Field string `json:"field"`
	// Fragment is plain text with the matches wrapped in <mark>.
	Fragment string `json:"fragment"`
	LineID   string `json:"lineId,omitempty"`
	At       string `json:"at,omitempty"`
}

type SearchHit struct {
	Talk       Talk        `json:"talk"`
	Highlights []Highlight `json:"highlights"`
}

// Talks stores talks in the talks index and searches them together with
// their notes and transcripts.
type Talks struct {
	es       *elasticsearch.TypedClient
	index    string
	captions string
	tracer   trace.Tracer
	created  metric.Int64Counter
	ended    metric.Int64Counter
	duration metric.Float64Histogram
}

func NewTalks(es *elasticsearch.TypedClient) (*Talks, error) {
	return newTalks(es, TalksIndex, CaptionsIndex)
}

func newTalks(es *elasticsearch.TypedClient, index, captionsIndex string) (*Talks, error) {
	created, err := otel.Meter(instrumentation).Int64Counter("talks.created",
		metric.WithDescription("Talks started from the web app"))
	if err != nil {
		return nil, err
	}
	ended, err := otel.Meter(instrumentation).Int64Counter("talks.ended",
		metric.WithDescription("Live talks ended from the web app"))
	if err != nil {
		return nil, err
	}
	duration, err := otel.Meter(instrumentation).Float64Histogram("talks.duration",
		metric.WithDescription("How long a talk was live before it ended"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(300, 900, 1800, 2700, 3600, 5400, 7200))
	if err != nil {
		return nil, err
	}
	return &Talks{
		es:       es,
		index:    index,
		captions: captionsIndex,
		tracer:   otel.Tracer(instrumentation),
		created:  created,
		ended:    ended,
		duration: duration,
	}, nil
}

// talkDoc shares the talks index with notesDoc. A talk's notes can be stored
// before the talk itself, so a document without started_at is not a talk.
type talkDoc struct {
	Title       string     `json:"title"`
	Speaker     string     `json:"speaker"`
	Description string     `json:"description"`
	StartedAt   time.Time  `json:"started_at"`
	EndedAt     *time.Time `json:"ended_at,omitempty"`
}

func (d talkDoc) talk(id string) Talk {
	status := TalkLive
	if d.EndedAt != nil {
		status = TalkFinished
	}
	return Talk{
		ID:          id,
		Title:       d.Title,
		Speaker:     d.Speaker,
		Description: d.Description,
		Status:      status,
		StartedAt:   d.StartedAt,
		EndedAt:     d.EndedAt,
	}
}

var isTalk = types.Query{Exists: &types.ExistsQuery{Field: "started_at"}}

// Create starts a live talk with an id made from its title, adding a number
// when another talk already has that id. It waits for a refresh so the talk
// is listed for everyone straight away.
func (t *Talks) Create(ctx context.Context, n NewTalk) (Talk, error) {
	ctx, span := t.tracer.Start(ctx, "talks.create")
	defer span.End()

	// Elasticsearch keeps milliseconds; truncating returns the time a later Get will.
	doc := talkDoc{
		Title:       n.Title,
		Speaker:     n.Speaker,
		Description: n.Description,
		StartedAt:   time.Now().UTC().Truncate(time.Millisecond),
	}
	base := slug(n.Title)
	id := base
	for try := 2; try <= maxSlugTries; try++ {
		_, err := t.es.Create(t.index, id).Document(doc).Refresh(refresh.Waitfor).Do(ctx)
		if err == nil {
			span.SetAttributes(attribute.String("talk.id", id), attribute.Int("talk.id_tries", try-1))
			t.created.Add(ctx, 1)
			return doc.talk(id), nil
		}
		if !isConflict(err) {
			span.RecordError(err)
			return Talk{}, fmt.Errorf("create talk: %w", err)
		}
		id = fmt.Sprintf("%s-%d", base, try)
	}
	return Talk{}, fmt.Errorf("create talk: no free id for %q", base)
}

// Get returns a talk, or nil when there is none with that id.
func (t *Talks) Get(ctx context.Context, id string) (*Talk, error) {
	ctx, span := t.tracer.Start(ctx, "talks.get", trace.WithAttributes(attribute.String("talk.id", id)))
	defer span.End()

	res, err := t.es.Get(t.index, id).Do(ctx)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("get talk: %w", err)
	}
	if !res.Found {
		return nil, nil
	}
	var doc talkDoc
	if err := json.Unmarshal(res.Source_, &doc); err != nil {
		return nil, fmt.Errorf("get talk: %w", err)
	}
	if doc.StartedAt.IsZero() {
		return nil, nil
	}
	talk := doc.talk(id)
	return &talk, nil
}

// Run inside Elasticsearch so two people ending a talk at once can't both set
// ended_at. A document without started_at holds only notes and isn't a talk.
const endTalkScript = `if (ctx._source.started_at == null || ctx._source.ended_at != null) { ctx.op = 'noop' } else { ctx._source.ended_at = params.now }`

// End finishes a live talk and returns it, or nil when there is no talk with
// that id. A talk that has already ended keeps its first end time. It waits
// for a refresh so the talk is listed as past for everyone straight away.
func (t *Talks) End(ctx context.Context, id string) (*Talk, error) {
	ctx, span := t.tracer.Start(ctx, "talks.end", trace.WithAttributes(attribute.String("talk.id", id)))
	defer span.End()

	now, err := json.Marshal(time.Now().UTC().Truncate(time.Millisecond))
	if err != nil {
		return nil, fmt.Errorf("end talk: %w", err)
	}
	res, err := t.es.Update(t.index, id).
		Script(&types.Script{Source: endTalkScript, Params: map[string]json.RawMessage{"now": now}}).
		SourceIncludes_("title", "speaker", "description", "started_at", "ended_at").
		Refresh(refresh.Waitfor).
		Do(ctx)
	if isNotFound(err) {
		return nil, nil
	}
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("end talk: %w", err)
	}
	if res.Get == nil {
		return nil, fmt.Errorf("end talk: no source in the response")
	}
	var doc talkDoc
	if err := json.Unmarshal(res.Get.Source_, &doc); err != nil {
		return nil, fmt.Errorf("end talk: %w", err)
	}
	if doc.StartedAt.IsZero() {
		return nil, nil
	}
	endedNow := res.Result == result.Updated
	span.SetAttributes(attribute.Bool("talk.already_ended", !endedNow))
	if endedNow {
		t.ended.Add(ctx, 1)
		t.duration.Record(ctx, doc.EndedAt.Sub(doc.StartedAt).Seconds())
	}
	talk := doc.talk(id)
	return &talk, nil
}

// List returns every talk, the most recently started first.
func (t *Talks) List(ctx context.Context) ([]Talk, error) {
	ctx, span := t.tracer.Start(ctx, "talks.list")
	defer span.End()

	res, err := t.es.Search().Index(t.index).
		Query(&isTalk).
		Sort(&types.SortOptions{SortOptions: map[string]types.FieldSort{"started_at": {Order: &sortorder.Desc}}}).
		Size(maxTalks).
		Do(ctx)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("list talks: %w", err)
	}
	talks, err := decodeTalks(res.Hits.Hits)
	if err != nil {
		return nil, fmt.Errorf("list talks: %w", err)
	}
	span.SetAttributes(attribute.Int("talks.count", len(talks)))
	return talks, nil
}

var (
	marks            = []string{"<mark>"}
	markEnds         = []string{"</mark>"}
	wholeField       = 0
	talkSearchFields = []string{"title^3", "speaker^2", "description", "summary", "notes"}
	// A title or a note is short enough to show whole.
	highlightedFields = []map[string]types.HighlightField{
		{"title": {NumberOfFragments: &wholeField}},
		{"description": {}},
		{"summary": {}},
		{"notes": {NumberOfFragments: &wholeField}},
	}
)

// Search finds talks whose details, notes or transcript mention query. Talks
// that match on their details or notes come first, in relevance order, then
// talks that only match on what was said.
func (t *Talks) Search(ctx context.Context, query string) ([]SearchHit, error) {
	ctx, span := t.tracer.Start(ctx, "talks.search", trace.WithAttributes(attribute.Int("search.query_length", len(query))))
	defer span.End()

	var (
		hits  []SearchHit
		lines transcriptMatches
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { hits, err = t.searchDetails(gctx, query); return err })
	g.Go(func() (err error) { lines, err = t.searchTranscripts(gctx, query); return err })
	if err := g.Wait(); err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("search talks: %w", err)
	}

	position := make(map[string]int, len(hits))
	for i, hit := range hits {
		position[hit.Talk.ID] = i
	}
	var transcriptOnly []string
	for _, id := range lines.talkIDs {
		if i, ok := position[id]; ok {
			hits[i].Highlights = append(hits[i].Highlights, lines.highlights[id]...)
		} else {
			transcriptOnly = append(transcriptOnly, id)
		}
	}
	if len(transcriptOnly) > 0 {
		talks, err := t.byID(ctx, transcriptOnly)
		if err != nil {
			span.RecordError(err)
			return nil, fmt.Errorf("search talks: %w", err)
		}
		for _, id := range transcriptOnly {
			if talk, ok := talks[id]; ok {
				hits = append(hits, SearchHit{Talk: talk, Highlights: lines.highlights[id]})
			}
		}
	}
	span.SetAttributes(attribute.Int("search.hits", len(hits)))
	return hits, nil
}

func (t *Talks) searchDetails(ctx context.Context, query string) ([]SearchHit, error) {
	res, err := t.es.Search().Index(t.index).
		Query(&types.Query{Bool: &types.BoolQuery{
			Must:   []types.Query{{MultiMatch: &types.MultiMatchQuery{Query: query, Fields: talkSearchFields}}},
			Filter: []types.Query{isTalk},
		}}).
		Highlight(&types.Highlight{PreTags: marks, PostTags: markEnds, Fields: highlightedFields}).
		Size(maxSearchHits).
		Do(ctx)
	if err != nil {
		return nil, err
	}
	talks, err := decodeTalks(res.Hits.Hits)
	if err != nil {
		return nil, err
	}
	hits := make([]SearchHit, len(talks))
	for i, talk := range talks {
		highlights := []Highlight{}
		for _, fields := range highlightedFields {
			for field := range fields {
				for _, fragment := range res.Hits.Hits[i].Highlight[field] {
					highlights = append(highlights, Highlight{Field: field, Fragment: fragment})
				}
			}
		}
		hits[i] = SearchHit{Talk: talk, Highlights: highlights}
	}
	return hits, nil
}

type transcriptMatches struct {
	talkIDs    []string
	highlights map[string][]Highlight
}

func (t *Talks) searchTranscripts(ctx context.Context, query string) (transcriptMatches, error) {
	name, size := "lines", transcriptHitsPerTalk
	res, err := t.es.Search().Index(t.captions).
		Query(&types.Query{Match: map[string]types.MatchQuery{"text": {Query: query}}}).
		Collapse(&types.FieldCollapse{Field: "talk_id", InnerHits: []types.InnerHits{{
			Name: &name,
			Size: &size,
			Sort: []types.SortCombinations{&types.SortOptions{SortOptions: map[string]types.FieldSort{"start_ms": {Order: &sortorder.Asc}}}},
			Highlight: &types.Highlight{
				PreTags:  marks,
				PostTags: markEnds,
				Fields:   []map[string]types.HighlightField{{"text": {NumberOfFragments: &wholeField}}},
			},
		}}}).
		Size(maxSearchHits).
		Do(ctx)
	if err != nil {
		return transcriptMatches{}, err
	}

	matches := transcriptMatches{highlights: map[string][]Highlight{}}
	for _, talkHit := range res.Hits.Hits {
		for _, line := range talkHit.InnerHits[name].Hits.Hits {
			var doc captionDoc
			if err := json.Unmarshal(line.Source_, &doc); err != nil {
				return transcriptMatches{}, err
			}
			fragments := line.Highlight["text"]
			if line.Id_ == nil || len(fragments) == 0 {
				continue
			}
			if _, seen := matches.highlights[doc.TalkID]; !seen {
				matches.talkIDs = append(matches.talkIDs, doc.TalkID)
			}
			matches.highlights[doc.TalkID] = append(matches.highlights[doc.TalkID], Highlight{
				Field:    "transcript",
				Fragment: fragments[0],
				LineID:   *line.Id_,
				At:       captions.FormatTime(time.UnixMilli(doc.StartMs)),
			})
		}
	}
	return matches, nil
}

func (t *Talks) byID(ctx context.Context, ids []string) (map[string]Talk, error) {
	res, err := t.es.Search().Index(t.index).
		Query(&types.Query{Bool: &types.BoolQuery{Filter: []types.Query{
			{Ids: &types.IdsQuery{Values: ids}},
			isTalk,
		}}}).
		Size(len(ids)).
		Do(ctx)
	if err != nil {
		return nil, err
	}
	talks, err := decodeTalks(res.Hits.Hits)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]Talk, len(talks))
	for _, talk := range talks {
		byID[talk.ID] = talk
	}
	return byID, nil
}

func decodeTalks(hits []types.Hit) ([]Talk, error) {
	talks := make([]Talk, 0, len(hits))
	for _, hit := range hits {
		var doc talkDoc
		if err := json.Unmarshal(hit.Source_, &doc); err != nil {
			return nil, err
		}
		id := ""
		if hit.Id_ != nil {
			id = *hit.Id_
		}
		talks = append(talks, doc.talk(id))
	}
	return talks, nil
}

func isConflict(err error) bool {
	return hasStatus(err, 409)
}

func isNotFound(err error) bool {
	return hasStatus(err, 404)
}

func hasStatus(err error, status int) bool {
	var esErr *types.ElasticsearchError
	return errors.As(err, &esErr) && esErr.Status == status
}

var notSlug = regexp.MustCompile(`[^a-z0-9]+`)

// slug makes a URL-safe talk id from a title: "Café: eBPF & You!" becomes
// "cafe-ebpf-you".
func slug(title string) string {
	stripAccents := transform.Chain(norm.NFKD, runes.Remove(runes.In(unicode.Mn)))
	folded, _, err := transform.String(stripAccents, title)
	if err != nil {
		folded = title
	}
	s := notSlug.ReplaceAllString(strings.ToLower(folded), "-")
	if len(s) > maxSlugLength {
		s = s[:maxSlugLength]
	}
	s = strings.Trim(s, "-")
	if s == "" {
		return "talk"
	}
	return s
}
