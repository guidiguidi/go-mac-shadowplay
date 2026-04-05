#import "capture.h"
#import <AVFoundation/AVFoundation.h>
#import <CoreMedia/CoreMedia.h>
#import <CoreVideo/CoreVideo.h>
#import <Foundation/Foundation.h>
#import <ScreenCaptureKit/ScreenCaptureKit.h>

extern void shadowplayOnSegmentClosed(const char *path);

static SCStream *g_stream;
static dispatch_queue_t g_writerQueue;

static volatile int g_manualRecording;
static volatile int g_rollingActive;

static AVAssetWriter *g_writer;
static AVAssetWriterInput *g_input;
static AVAssetWriterInputPixelBufferAdaptor *g_adaptor;

static NSString *g_rollDir;
static double g_segmentSeconds;
static double g_maxBufferSeconds;
static CGSize g_rollSize;

static AVAssetWriter *g_rollWriter;
static AVAssetWriterInput *g_rollInput;
static AVAssetWriterInputPixelBufferAdaptor *g_rollAdaptor;

static NSMutableArray<NSString *> *g_rollFiles;
static NSLock *g_rollFilesLock;
static dispatch_source_t g_rollRotateTimer;

@interface SPStreamDelegate : NSObject <SCStreamOutput>
@property(nonatomic, strong) AVAssetWriter *writer;
@property(nonatomic, strong) AVAssetWriterInput *input;
@property(nonatomic, strong) AVAssetWriterInputPixelBufferAdaptor *adaptor;
@property(nonatomic, assign) int sessionStarted;
@end

@implementation SPStreamDelegate

- (void)stream:(SCStream *)stream
    didOutputSampleBuffer:(CMSampleBufferRef)sampleBuffer
                   ofType:(SCStreamOutputType)type {
  if (type != SCStreamOutputTypeScreen) {
    return;
  }

  CVPixelBufferRef pixelBuffer = CMSampleBufferGetImageBuffer(sampleBuffer);
  if (!pixelBuffer) {
    return;
  }

  CMTime pts = CMSampleBufferGetPresentationTimeStamp(sampleBuffer);

  AVAssetWriter *writer = self.writer;
  AVAssetWriterInput *input = self.input;
  AVAssetWriterInputPixelBufferAdaptor *adaptor = self.adaptor;
  if (!writer || !input || !adaptor) {
    return;
  }

  dispatch_async(g_writerQueue, ^{
    if (writer.status == AVAssetWriterStatusFailed) {
      return;
    }
    if (!self.sessionStarted) {
      [writer startWriting];
      [writer startSessionAtSourceTime:pts];
      self.sessionStarted = 1;
    }
    if (input.readyForMoreMediaData) {
      [adaptor appendPixelBuffer:pixelBuffer withPresentationTime:pts];
    }
  });
}

@end

static SPStreamDelegate *g_delegate;

static void sp_reset_manual_writer(void) {
  g_writer = nil;
  g_input = nil;
  g_adaptor = nil;
}

static int sp_begin_segment_writer(NSString *path, CGSize size, int32_t bitrate,
                                   AVAssetWriter **outW, AVAssetWriterInput **outIn,
                                   AVAssetWriterInputPixelBufferAdaptor **outAd, int *sessionStarted) {
  NSURL *url = [NSURL fileURLWithPath:path];
  [[NSFileManager defaultManager] removeItemAtURL:url error:nil];

  NSError *nwErr = nil;
  AVAssetWriter *writer = [[AVAssetWriter alloc] initWithURL:url fileType:AVFileTypeQuickTimeMovie error:&nwErr];
  if (!writer || nwErr) {
    return -10;
  }

  NSDictionary *compression = @{
    AVVideoAverageBitRateKey : @(bitrate),
  };
  NSDictionary *settings = @{
    AVVideoCodecKey : AVVideoCodecTypeH264,
    AVVideoWidthKey : @(size.width),
    AVVideoHeightKey : @(size.height),
    AVVideoCompressionPropertiesKey : compression,
  };

  AVAssetWriterInput *input =
      [[AVAssetWriterInput alloc] initWithMediaType:AVMediaTypeVideo outputSettings:settings];
  input.expectsMediaDataInRealTime = YES;

  NSDictionary *pix = @{
    (NSString *)kCVPixelBufferPixelFormatTypeKey : @(kCVPixelFormatType_420YpCbCr8BiPlanarVideoRange),
  };
  AVAssetWriterInputPixelBufferAdaptor *adaptor =
      [[AVAssetWriterInputPixelBufferAdaptor alloc] initWithAssetWriterInput:input
                                                  sourcePixelBufferAttributes:pix];

  if (![writer canAddInput:input]) {
    return -11;
  }
  [writer addInput:input];

  *outW = writer;
  *outIn = input;
  *outAd = adaptor;
  *sessionStarted = 0;
  return 0;
}

static int sp_run_shareable_content(CGSize *outSize, SCDisplay **outDisplay) {
  __block SCShareableContent *content = nil;
  __block NSError *err = nil;
  dispatch_semaphore_t sem = dispatch_semaphore_create(0);
  [SCShareableContent getShareableContentWithCompletionHandler:^(SCShareableContent *_Nullable c,
                                                                  NSError *_Nullable e) {
    content = c;
    err = e;
    dispatch_semaphore_signal(sem);
  }];
  dispatch_semaphore_wait(sem, DISPATCH_TIME_FOREVER);
  if (err || !content || content.displays.count == 0) {
    return -1;
  }
  SCDisplay *display = content.displays.firstObject;
  *outSize = CGSizeMake(display.width, display.height);
  *outDisplay = display;
  return 0;
}

static int sp_start_stream(SCContentFilter *filter, CGSize pixelSize, int32_t fps, int32_t bitrate,
                           SPStreamDelegate *delegate) {
  SCStreamConfiguration *cfg = [[SCStreamConfiguration alloc] init];
  cfg.width = (NSInteger)pixelSize.width;
  cfg.height = (NSInteger)pixelSize.height;
  cfg.minimumFrameInterval = CMTimeMake(1, fps);
  cfg.pixelFormat = kCVPixelFormatType_420YpCbCr8BiPlanarVideoRange;
  cfg.showsCursor = YES;
  cfg.capturesAudio = NO;

  NSError *addErr = nil;
  SCStream *stream = [[SCStream alloc] initWithFilter:filter configuration:cfg delegate:nil];

  [stream addStreamOutput:delegate type:SCStreamOutputTypeScreen sampleHandlerQueue:g_writerQueue error:&addErr];
  if (addErr) {
    return -3;
  }

  g_stream = stream;
  g_delegate = delegate;

  dispatch_semaphore_t sem = dispatch_semaphore_create(0);
  __block int startErr = 0;
  [stream startCaptureWithCompletionHandler:^(NSError *_Nullable error) {
    if (error) {
      startErr = -4;
    }
    dispatch_semaphore_signal(sem);
  }];

  dispatch_semaphore_wait(sem, DISPATCH_TIME_FOREVER);
  return startErr;
}

int sp_capture_start(const char *output_path) {
  if (g_manualRecording || g_rollingActive) {
    return -100;
  }
  if (!output_path) {
    return -101;
  }

  if (!g_writerQueue) {
    g_writerQueue = dispatch_queue_create("com.guidiguidi.shadowplay.writer", DISPATCH_QUEUE_SERIAL);
  }

  CGSize size;
  SCDisplay *display = nil;
  int rc = sp_run_shareable_content(&size, &display);
  if (rc != 0) {
    return rc;
  }

  SCContentFilter *filter = [[SCContentFilter alloc] initWithDisplay:display excludingWindows:@[]];

  NSString *path = [NSString stringWithUTF8String:output_path];
  AVAssetWriter *writer = nil;
  AVAssetWriterInput *input = nil;
  AVAssetWriterInputPixelBufferAdaptor *adaptor = nil;
  int sess = 0;
  rc = sp_begin_segment_writer(path, size, 12000000, &writer, &input, &adaptor, &sess);
  if (rc != 0) {
    return rc;
  }

  SPStreamDelegate *del = [[SPStreamDelegate alloc] init];
  del.writer = writer;
  del.input = input;
  del.adaptor = adaptor;
  del.sessionStarted = 0;

  g_writer = writer;
  g_input = input;
  g_adaptor = adaptor;

  rc = sp_start_stream(filter, size, 60, 12000000, del);
  if (rc != 0) {
    sp_reset_manual_writer();
    g_stream = nil;
    g_delegate = nil;
    return rc;
  }

  g_manualRecording = 1;
  return 0;
}

int sp_capture_stop(void) {
  if (!g_manualRecording) {
    return 0;
  }
  g_manualRecording = 0;

  SCStream *stream = g_stream;
  SPStreamDelegate *del = g_delegate;
  AVAssetWriter *writer = g_writer;
  AVAssetWriterInput *input = g_input;

  g_stream = nil;
  g_delegate = nil;
  sp_reset_manual_writer();

  dispatch_semaphore_t sem = dispatch_semaphore_create(0);
  [stream stopCaptureWithCompletionHandler:^(NSError *_Nullable error) {
    dispatch_semaphore_signal(sem);
  }];
  dispatch_semaphore_wait(sem, DISPATCH_TIME_FOREVER);

  dispatch_semaphore_t wsem = dispatch_semaphore_create(0);
  dispatch_async(g_writerQueue, ^{
    [input markAsFinished];
    [writer finishWritingWithCompletionHandler:^{
      dispatch_semaphore_signal(wsem);
    }];
  });
  dispatch_semaphore_wait(wsem, DISPATCH_TIME_FOREVER);

  del.writer = nil;
  del.input = nil;
  del.adaptor = nil;

  return 0;
}

int sp_capture_is_recording(void) { return g_manualRecording ? 1 : 0; }

static void sp_roll_trim_files_locked(void) {
  NSInteger maxCount = (NSInteger)ceil(g_maxBufferSeconds / g_segmentSeconds) + 2;
  if (maxCount < 2) {
    maxCount = 2;
  }
  while ((NSInteger)g_rollFiles.count > maxCount) {
    NSString *first = g_rollFiles.firstObject;
    [[NSFileManager defaultManager] removeItemAtPath:first error:nil];
    [g_rollFiles removeObjectAtIndex:0];
  }
}

static void sp_roll_open_new_segment(void) {
  NSUUID *uuid = [NSUUID UUID];
  NSString *name = [NSString stringWithFormat:@"seg_%@.mov", uuid.UUIDString];
  NSString *path = [g_rollDir stringByAppendingPathComponent:name];

  AVAssetWriter *writer = nil;
  AVAssetWriterInput *input = nil;
  AVAssetWriterInputPixelBufferAdaptor *adaptor = nil;
  int sess = 0;
  int rc = sp_begin_segment_writer(path, g_rollSize, 12000000, &writer, &input, &adaptor, &sess);
  if (rc != 0) {
    return;
  }

  g_rollWriter = writer;
  g_rollInput = input;
  g_rollAdaptor = adaptor;

  g_delegate.writer = writer;
  g_delegate.input = input;
  g_delegate.adaptor = adaptor;
  g_delegate.sessionStarted = 0;
}

static void sp_roll_finish_segment(void (^then)(void)) {
  AVAssetWriter *w = g_rollWriter;
  AVAssetWriterInput *in = g_rollInput;
  NSString *closedPath = w.outputURL.path;

  g_rollWriter = nil;
  g_rollInput = nil;
  g_rollAdaptor = nil;
  g_delegate.writer = nil;
  g_delegate.input = nil;
  g_delegate.adaptor = nil;

  if (!w || !in) {
    if (then) {
      then();
    }
    return;
  }

  dispatch_async(g_writerQueue, ^{
    [in markAsFinished];
    [w finishWritingWithCompletionHandler:^{
      if (closedPath.length > 0) {
        [g_rollFilesLock lock];
        [g_rollFiles addObject:closedPath];
        shadowplayOnSegmentClosed([closedPath UTF8String]);
        sp_roll_trim_files_locked();
        [g_rollFilesLock unlock];
      }
      if (then) {
        dispatch_async(g_writerQueue, ^{
          then();
        });
      }
    }];
  });
}

int sp_rolling_start(const char *output_dir, double segment_seconds, double max_buffer_seconds) {
  if (g_manualRecording || g_rollingActive) {
    return -200;
  }
  if (!output_dir || segment_seconds <= 0 || max_buffer_seconds <= 0) {
    return -201;
  }

  if (!g_writerQueue) {
    g_writerQueue = dispatch_queue_create("com.guidiguidi.shadowplay.writer", DISPATCH_QUEUE_SERIAL);
  }

  NSString *dir = [NSString stringWithUTF8String:output_dir];
  [[NSFileManager defaultManager] createDirectoryAtPath:dir withIntermediateDirectories:YES attributes:nil error:nil];

  CGSize size;
  SCDisplay *display = nil;
  int rc = sp_run_shareable_content(&size, &display);
  if (rc != 0) {
    return rc;
  }

  g_rollDir = dir;
  g_segmentSeconds = segment_seconds;
  g_maxBufferSeconds = max_buffer_seconds;
  g_rollSize = size;
  g_rollFiles = [NSMutableArray array];
  g_rollFilesLock = [[NSLock alloc] init];

  SCContentFilter *filter = [[SCContentFilter alloc] initWithDisplay:display excludingWindows:@[]];

  g_delegate = [[SPStreamDelegate alloc] init];
  sp_roll_open_new_segment();

  rc = sp_start_stream(filter, size, 60, 12000000, g_delegate);
  if (rc != 0) {
    g_rollDir = nil;
    g_delegate = nil;
    g_stream = nil;
    return rc;
  }

  g_rollingActive = 1;

  if (g_rollRotateTimer) {
    dispatch_source_cancel(g_rollRotateTimer);
    g_rollRotateTimer = nil;
  }
  dispatch_queue_t tq = dispatch_get_global_queue(QOS_CLASS_USER_INITIATED, 0);
  g_rollRotateTimer = dispatch_source_create(DISPATCH_SOURCE_TYPE_TIMER, 0, 0, tq);
  uint64_t nsec = (uint64_t)(segment_seconds * NSEC_PER_SEC);
  dispatch_source_set_timer(g_rollRotateTimer, dispatch_time(DISPATCH_TIME_NOW, nsec), nsec,
                             (uint64_t)(0.05 * NSEC_PER_SEC));
  dispatch_source_set_event_handler(g_rollRotateTimer, ^{
    if (!g_rollingActive) {
      return;
    }
    dispatch_async(g_writerQueue, ^{
      if (!g_rollingActive) {
        return;
      }
      sp_roll_finish_segment(^{
        if (g_rollingActive) {
          sp_roll_open_new_segment();
        }
      });
    });
  });
  dispatch_resume(g_rollRotateTimer);

  return 0;
}

int sp_rolling_stop(void) {
  if (!g_rollingActive) {
    return 0;
  }
  g_rollingActive = 0;

  if (g_rollRotateTimer) {
    dispatch_source_cancel(g_rollRotateTimer);
    g_rollRotateTimer = nil;
  }

  SCStream *stream = g_stream;
  g_stream = nil;

  dispatch_semaphore_t sem = dispatch_semaphore_create(0);
  [stream stopCaptureWithCompletionHandler:^(NSError *_Nullable error) {
    dispatch_semaphore_signal(sem);
  }];
  dispatch_semaphore_wait(sem, DISPATCH_TIME_FOREVER);

  dispatch_semaphore_t wsem = dispatch_semaphore_create(0);
  sp_roll_finish_segment(^{
    dispatch_semaphore_signal(wsem);
  });
  dispatch_semaphore_wait(wsem, DISPATCH_TIME_FOREVER);

  g_delegate.writer = nil;
  g_delegate.input = nil;
  g_delegate.adaptor = nil;
  g_delegate = nil;

  g_rollDir = nil;
  return 0;
}

int sp_rolling_is_active(void) { return g_rollingActive ? 1 : 0; }

int sp_rolling_export_last(const char *output_path, double duration_seconds) {
  if (!output_path || duration_seconds <= 0) {
    return -300;
  }

  [g_rollFilesLock lock];
  NSArray<NSString *> *files = [g_rollFiles copy];
  [g_rollFilesLock unlock];

  if (files.count == 0) {
    return -301;
  }

  AVMutableComposition *comp = [AVMutableComposition composition];
  AVMutableCompositionTrack *vtrack =
      [comp addMutableTrackWithMediaType:AVMediaTypeVideo preferredTrackID:kCMPersistentTrackID_Invalid];

  CMTime cursor = kCMTimeZero;
  for (NSString *p in files) {
    NSURL *url = [NSURL fileURLWithPath:p];
    AVURLAsset *asset = [AVURLAsset URLAssetWithURL:url options:nil];
    AVAssetTrack *t = [[asset tracksWithMediaType:AVMediaTypeVideo] firstObject];
    if (!t) {
      continue;
    }
    CMTime dur = asset.duration;
    if (CMTIME_COMPARE_INLINE(dur, ==, kCMTimeInvalid)) {
      continue;
    }
    CMTimeRange range = CMTimeRangeMake(kCMTimeZero, dur);
    NSError *err = nil;
    [vtrack insertTimeRange:range ofTrack:t atTime:cursor error:&err];
    if (err) {
      return -303;
    }
    cursor = CMTimeAdd(cursor, dur);
  }

  if (CMTIME_COMPARE_INLINE(cursor, ==, kCMTimeInvalid) || CMTimeCompare(cursor, kCMTimeZero) <= 0) {
    return -302;
  }

  CMTime want = CMTimeMakeWithSeconds(duration_seconds, 600);
  CMTime start = kCMTimeZero;
  if (CMTimeCompare(cursor, want) > 0) {
    start = CMTimeSubtract(cursor, want);
  }
  CMTimeRange exportRange = CMTimeRangeMake(start, CMTimeSubtract(cursor, start));

  NSString *out = [NSString stringWithUTF8String:output_path];
  NSURL *outURL = [NSURL fileURLWithPath:out];
  [[NSFileManager defaultManager] removeItemAtURL:outURL error:nil];

  AVAssetExportSession *exporter =
      [[AVAssetExportSession alloc] initWithAsset:comp presetName:AVAssetExportPresetHighestQuality];
  exporter.outputURL = outURL;
  exporter.outputFileType = AVFileTypeMPEG4;
  exporter.timeRange = exportRange;

  dispatch_semaphore_t sem = dispatch_semaphore_create(0);
  [exporter exportAsynchronouslyWithCompletionHandler:^{
    dispatch_semaphore_signal(sem);
  }];
  dispatch_semaphore_wait(sem, DISPATCH_TIME_FOREVER);

  if (exporter.status != AVAssetExportSessionStatusCompleted) {
    return -306;
  }
  return 0;
}
