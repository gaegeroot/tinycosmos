package com.gaegeroot.tinycosmos

import android.appwidget.AppWidgetManager
import android.appwidget.AppWidgetProvider
import android.content.Context
import android.widget.RemoteViews

/** Static home screen widget; all content comes from res/layout/tiny_cosmos_widget.xml. */
class TinyCosmosWidgetProvider : AppWidgetProvider() {
    override fun onUpdate(
        context: Context,
        appWidgetManager: AppWidgetManager,
        appWidgetIds: IntArray,
    ) {
        for (appWidgetId in appWidgetIds) {
            val views = RemoteViews(context.packageName, R.layout.tiny_cosmos_widget)
            appWidgetManager.updateAppWidget(appWidgetId, views)
        }
    }
}
