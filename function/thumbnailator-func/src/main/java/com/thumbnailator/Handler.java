package thumbnailator;

import com.amazonaws.services.lambda.runtime.Context;
import com.amazonaws.services.lambda.runtime.RequestHandler;
import com.amazonaws.services.lambda.runtime.LambdaLogger;

import java.util.Map;
import java.util.HashMap;

import java.lang.Error;
import java.awt.geom.AffineTransform;
import java.awt.image.AffineTransformOp;
import java.net.URL;
import java.awt.image.BufferedImage;
import java.awt.image.WritableRaster;
import java.io.ByteArrayInputStream;
import java.io.ByteArrayOutputStream;
import java.io.InputStream;
import java.awt.image.ColorModel;
import javax.imageio.ImageIO;

public class Handler implements RequestHandler<Map<String,String>, String>{
    static boolean exit; 
    static double scale;
    static BufferedImage image;
    static byte[] binaryImage;
    private int reqCount;

    static {
        try {
            ImageIO.setUseCache(false); // We don't want to cache things out for experimento purposes.
            
            scale = Double.parseDouble(System.getenv("scale"));
            
            // Reading raw bytes of the image.
            URL url = new URL(System.getenv("image_url"));
            image = ImageIO.read(url);
            int contentLength = url.openConnection().getContentLength();
            
            ByteArrayOutputStream output = new ByteArrayOutputStream();
            InputStream inputStream = url.openStream();
            int n = 0;
            byte[] buffer = new byte[contentLength];
            while (-1 != (n = inputStream.read(buffer))) {
                output.write(buffer, 0, n);
            }
            
            binaryImage = output.toByteArray();
            int imageSize = binaryImage.length;
            if (imageSize != contentLength) {
                throw new RuntimeException(
                    String.format("Size of the downloaded image %d is different from the content length %d",
                    imageSize, contentLength));
                }
                inputStream.close();
            } catch (Exception e) {
            e.printStackTrace();
            exit = true;
        }
    }

    public String handleRequest(Map<String,String> event, Context context)
    {
        if (exit) {
            System.exit(1);
        }

        long before = System.nanoTime();
        String err = callFunction();
        long after = System.nanoTime();

        String output = err + System.lineSeparator();
        if (err.length() == 0) {
            long serviceTime = ((long) (after - before)); // service time in nanoseconds
            output = Long.toString(serviceTime);
        } else {
            // do nothing yet, but should somehow return 503
        }

        String response = new String(output);
        return response;
    }

    public String callFunction() {
        String err = "";
        try {
            // avoid that the return from method escape to stack
            byte[] arr = simulateImageDownload();
            
            AffineTransform transform = AffineTransform.getScaleInstance(scale, scale); 
            AffineTransformOp op = new AffineTransformOp(transform, AffineTransformOp.TYPE_BILINEAR); 
            op.filter(image, null).flush();

            // make sure that it will not escape to stack
            for (int i = 0; i < arr.length; i++) {
                arr[i] = binaryImage[i];
            }

        } catch (Exception e) {
            err = e.toString() + System.lineSeparator()
            + e.getCause() + System.lineSeparator()
            + e.getMessage();
            e.printStackTrace();

        } catch (Error e) {
            err = e.toString() + System.lineSeparator()
            + e.getCause() + System.lineSeparator()
            + e.getMessage();
            e.printStackTrace();
        }
        return err;
    }

    private byte[] simulateImageDownload() {
        // This copy aims to simulate the effect of downloading the binary image from an
        // URL, but without having to deal with the variance imposed by network
        // transmission churn.
        byte[] rawCopy = new byte[binaryImage.length];
        for (int i = 0; i < rawCopy.length; i++) {
            rawCopy[i] = binaryImage[i];
        }
        return rawCopy;
    }

}
