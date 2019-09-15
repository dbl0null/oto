// Copyright 2019 The Oto Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// +build darwin,ios

package oto

// #cgo CFLAGS: -x objective-c
// #cgo LDFLAGS: -framework Foundation -framework AVFoundation
//
// #import <AVFoundation/AVFoundation.h>
// #import <AudioToolbox/AudioToolbox.h>
//
// @interface OtoInterruptObserver : NSObject {
// }
//
// @property (nonatomic) AudioUnit audioUnit;
//
// - (void) onAudioSessionEvent: (NSNotification*)notification;
//
// @end
//
// @implementation OtoInterruptObserver {
//   AudioUnit _audioUnit;
// }
//
// - (void) onAudioSessionEvent: (NSNotification *) notification
// {
//   if (![notification.name isEqualToString:AVAudioSessionInterruptionNotification]) {
//     return;
//   }
// 
//   NSObject* value = [notification.userInfo valueForKey:AVAudioSessionInterruptionTypeKey];
//   AVAudioSessionInterruptionType interruptionType = [(NSNumber*)value intValue];
//   switch (interruptionType) {
//   case AVAudioSessionInterruptionTypeBegan:
//     AudioOutputUnitStop([self audioUnit]);
//     break;
//   case AVAudioSessionInterruptionTypeEnded:
//     AudioOutputUnitStart([self audioUnit]);
//     break;
//   default:
//     NSAssert(NO, @"unexpected AVAudioSessionInterruptionType: %d", interruptionType);
//     break;
//   }
// }
//
// @end
//
// // Handle interruption events, or Siri would stop the audio (#80).
// static void setNotificationHandler(AudioUnit audioUnit) {
//   AVAudioSession* session = [AVAudioSession sharedInstance];
//   OtoInterruptObserver* observer = [[OtoInterruptObserver alloc] init];
//   observer.audioUnit = audioUnit;
//   [[NSNotificationCenter defaultCenter] addObserver: observer
//                                            selector: @selector(onAudioSessionEvent:)
//                                                name: AVAudioSessionInterruptionNotification
//                                              object: session];
// }
import "C"

func setNotificationHandler(driver *driver) {
	C.setNotificationHandler(driver.audioUnit)
}

func componentSubType() C.OSType {
	return C.kAudioUnitSubType_RemoteIO
}
