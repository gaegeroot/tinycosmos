import SwiftUI
import WidgetKit

/// Static home screen widget showing a hard-coded question. No app data, no reloads.

struct TinyCosmosEntry: TimelineEntry {
    let date: Date
}

struct TinyCosmosProvider: TimelineProvider {
    func placeholder(in context: Context) -> TinyCosmosEntry {
        TinyCosmosEntry(date: Date())
    }

    func getSnapshot(in context: Context, completion: @escaping (TinyCosmosEntry) -> Void) {
        completion(TinyCosmosEntry(date: Date()))
    }

    func getTimeline(in context: Context, completion: @escaping (Timeline<TinyCosmosEntry>) -> Void) {
        completion(Timeline(entries: [TinyCosmosEntry(date: Date())], policy: .never))
    }
}

private extension Color {
    static let cosmosBackground = Color(red: 0x0E / 255, green: 0x13 / 255, blue: 0x30 / 255)
    static let cosmosLabel = Color(red: 0xF2 / 255, green: 0xC4 / 255, blue: 0x6B / 255)
    static let cosmosSnippet = Color(red: 0xF4 / 255, green: 0xF5 / 255, blue: 0xFB / 255)
}

struct TinyCosmosWidgetView: View {
    let entry: TinyCosmosEntry

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text("This week's question")
                .font(.caption)
                .fontWeight(.semibold)
                .foregroundColor(.cosmosLabel)
            Text("What are you excited about this week?")
                .font(.headline)
                .foregroundColor(.cosmosSnippet)
            Spacer(minLength: 0)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
        .containerBackground(for: .widget) {
            Color.cosmosBackground
        }
    }
}

@main
struct TinyCosmosWidget: Widget {
    let kind = "TinyCosmosWidget"

    var body: some WidgetConfiguration {
        StaticConfiguration(kind: kind, provider: TinyCosmosProvider()) { entry in
            TinyCosmosWidgetView(entry: entry)
        }
        .configurationDisplayName("Tiny Cosmos")
        .description("Shows this week's check-in question.")
        .supportedFamilies([.systemSmall])
    }
}
