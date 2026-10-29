#import <AVFoundation/AVFoundation.h>
#import <Foundation/Foundation.h>

// ARKit does not use Unity's WebCamTexture feature. In an AR-only player,
// Unity's generated UNITY_USES_WEBCAM=0 can compile its webcam permission API
// into stubs. Ask the OS directly without opening a second camera capture.
extern "C" int FN_CameraAuthorizationStatus()
{
    switch ([AVCaptureDevice authorizationStatusForMediaType:AVMediaTypeVideo])
    {
        case AVAuthorizationStatusNotDetermined: return 0;
        case AVAuthorizationStatusAuthorized: return 1;
        case AVAuthorizationStatusDenied: return 2;
        case AVAuthorizationStatusRestricted: return 3;
    }
    return 3;
}

extern "C" void FN_RequestCameraAuthorization()
{
    dispatch_async(dispatch_get_main_queue(), ^{
        if ([AVCaptureDevice authorizationStatusForMediaType:AVMediaTypeVideo] == AVAuthorizationStatusNotDetermined)
            [AVCaptureDevice requestAccessForMediaType:AVMediaTypeVideo completionHandler:^(BOOL granted) {
                // C# observes the resulting OS state; no camera image is read.
            }];
    });
}
