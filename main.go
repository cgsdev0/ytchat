package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"

	"ytchat/youtube"
)

func main() {
	accessToken := os.Getenv("YOUTUBE_ACCESS_TOKEN")
	channelID := os.Getenv("CHANNEL_ID")

	if accessToken == "" || channelID == "" {
		log.Fatal("YOUTUBE_ACCESS_TOKEN and CHANNEL_ID must be set")
	}

	creds := credentials.NewClientTLSFromCert(nil, "")
	conn, err := grpc.Dial(
		"youtube.googleapis.com:443",
		grpc.WithTransportCredentials(creds),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	client := youtube.NewYouTubeServiceClient(conn)

	ctx := metadata.AppendToOutgoingContext(
		context.Background(),
		"authorization", "Bearer "+accessToken,
	)

	// --- Step 1: Resolve liveChatId from channel ---
	videosResp, err := client.VideosList(ctx, &youtube.VideosListRequest{
		Part: []string{
			"liveStreamingDetails",
		},
		ChannelId: channelID,
		EventType: youtube.VideoEventType_LIVE,
		Type: []string{
			"video",
		},
		MaxResults: 1,
	})
	if err != nil {
		log.Fatal(err)
	}

	if len(videosResp.Items) == 0 ||
		videosResp.Items[0].LiveStreamingDetails == nil ||
		videosResp.Items[0].LiveStreamingDetails.ActiveLiveChatId == "" {
		log.Fatal("no active live chat found for channel")
	}

	liveChatID := videosResp.Items[0].
		LiveStreamingDetails.
		ActiveLiveChatId

	// --- Step 2: Stream live chat messages ---
	stream, err := client.LiveChatMessagesStreamList(ctx,
		&youtube.LiveChatMessagesStreamListRequest{
			LiveChatId: liveChatID,
			Part: []string{
				"snippet",
				"authorDetails",
			},
		})
	if err != nil {
		log.Fatal(err)
	}

	for {
		resp, err := stream.Recv()
		if err != nil {
			log.Fatal(err)
		}

		for _, item := range resp.Items {
			fmt.Printf(
				"[%s] %s: %s\n",
				item.Snippet.PublishedAt,
				item.AuthorDetails.DisplayName,
				item.Snippet.DisplayMessage,
			)
		}
	}
}
